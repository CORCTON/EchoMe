package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
	"golang.org/x/sync/errgroup"
)

// ASRConfig 定义ASR参数配置
type ASRConfig struct {
	Model         string   `json:"model"`                    // 模型名称
	Format        string   `json:"format"`                   // 音频格式
	SampleRate    int      `json:"sample_rate"`              // 采样率
	LanguageHints []string `json:"language_hints,omitempty"` // 语言提示
}

// DefaultASRConfig 返回Paraformer模型的默认配置
func DefaultASRConfig() ASRConfig {
	return ASRConfig{
		Model:      "paraformer-realtime-v2",
		Format:     "pcm",
		SampleRate: 16000,
	}
}

// HandleASR 通过阿里云WebSocket处理语音转文本
func (client *AliClient) HandleASR(ctx context.Context, clientWS *websocket.Conn, config domain.ASRConfig) error {
	// 转换配置类型
	asrConfig := ASRConfig{
		Model:         config.Model,
		Format:        config.Format,
		SampleRate:    config.SampleRate,
		LanguageHints: config.LanguageHints,
	}

	aliWS, err := connectToAliyunASR(client.apiKey, client.endPoint)
	if err != nil {
		return err
	}
	defer aliWS.Close()

	taskID, err := startASRTTask(aliWS, asrConfig)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	// 将客户端音频转发到阿里云
	g.Go(func() error {
		return forwardAudioToAliyun(ctx, clientWS, aliWS, taskID)
	})

	// 从阿里云读取结果并处理
	g.Go(func() error {
		return handleASRResults(ctx, aliWS, clientWS, client, taskID)
	})

	return g.Wait()
}

// connectToAliyunASR 连接到阿里云ASR WebSocket
func connectToAliyunASR(apiKey string, endpoint string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+apiKey)
	// 从配置的endpoint构建WebSocket URL
	wsURL := strings.Replace(endpoint, "https://", "wss://", 1) + "/api-ws/v1/inference"
	ws, _, err := dialer.Dial(wsURL, headers)
	return ws, err
}

// startASRTTask 发送启动任务指令并等待确认，返回任务ID
func startASRTTask(aliWS *websocket.Conn, config ASRConfig) (string, error) {
	taskID := uuid.New().String()
	runTask := map[string]interface{}{
		"header": map[string]interface{}{
			"action":    "run-task",
			"task_id":   taskID,
			"streaming": "duplex",
		},
		"payload": map[string]interface{}{
			"task_group": "audio",
			"task":       "asr",
			"function":   "recognition",
			"model":      config.Model,
			"parameters": map[string]interface{}{
				"format":                     config.Format,
				"sample_rate":                config.SampleRate,
				"disfluency_removal_enabled": false,
				"language_hints":             config.LanguageHints,
			},
			"input": map[string]interface{}{},
		},
	}
	if err := aliWS.WriteJSON(runTask); err != nil {
		return "", err
	}

	var started bool
	for !started {
		_, msg, err := aliWS.ReadMessage()
		if err != nil {
			return "", err
		}
		var response map[string]interface{}
		if err := json.Unmarshal(msg, &response); err != nil {
			continue
		}
		header, ok := response["header"].(map[string]interface{})
		if !ok {
			continue
		}
		if event, ok := header["event"].(string); ok && event == "task-started" {
			started = true
		}
	}
	return taskID, nil
}

// forwardAudioToAliyun 将客户端音频转发到阿里云，并在结束时发送finish-task
func forwardAudioToAliyun(ctx context.Context, clientWS, aliWS *websocket.Conn, taskID string) error {
	defer func() {
		finishTask := map[string]interface{}{
			"header": map[string]interface{}{
				"action":    "finish-task",
				"task_id":   taskID,
				"streaming": "duplex",
			},
			"payload": map[string]interface{}{
				"input": map[string]interface{}{},
			},
		}
		_ = aliWS.WriteJSON(finishTask)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			mt, data, err := clientWS.ReadMessage()
			if err != nil {
				return nil // 客户端断开
			}
			if mt == websocket.BinaryMessage {
				if err := aliWS.WriteMessage(websocket.BinaryMessage, data); err != nil {
					return fmt.Errorf("转发音频到阿里云失败: %w", err)
				}
			}
		}
	}
}

// handleASRResults 从阿里云读取结果，生成AI响应并发送给客户端
func handleASRResults(ctx context.Context, aliWS, clientWS *websocket.Conn, client *AliClient, taskID string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, msg, err := aliWS.ReadMessage()
			if err != nil {
				return nil // 阿里云连接断开
			}
			var response map[string]interface{}
			if err := json.Unmarshal(msg, &response); err != nil {
				continue
			}
			header, ok := response["header"].(map[string]interface{})
			if !ok {
				continue
			}
			event, ok := header["event"].(string)
			if !ok {
				continue
			}
			if event == "result-generated" {
				payload, ok := response["payload"].(map[string]interface{})
				if !ok {
					continue
				}
				output, ok := payload["output"].(map[string]interface{})
				if !ok {
					continue
				}
				sentence, ok := output["sentence"].(map[string]interface{})
				if !ok {
					continue
				}
				text, ok := sentence["text"].(string)
				if !ok {
					continue
				}
				aiResponse, err := client.GenerateResponse(ctx, text, "")
				if err != nil {
					// 记录错误但继续处理
					fmt.Printf("AI服务错误: %v\n", err)
					continue
				}
				if err := clientWS.WriteJSON(map[string]string{"type": "asr_text", "text": aiResponse}); err != nil {
					return fmt.Errorf("发送AI响应到客户端失败: %w", err)
				}
			} else if event == "task-finished" || event == "task-failed" {
				return nil
			}
		}
	}
}