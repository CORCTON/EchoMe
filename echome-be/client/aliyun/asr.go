package aliyun

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
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
	// 连接到阿里云ASR WebSocket
	dialer := websocket.Dialer{}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+client.apiKey)
	aliWS, _, err := dialer.Dial("wss://dashscope.aliyuncs.com/api-ws/v1/inference", headers)
	if err != nil {
		return err
	}
	defer aliWS.Close()

	// 生成任务ID
	taskID := uuid.New().String()

	// 发送启动任务指令
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
			"model":      asrConfig.Model,
			"parameters": map[string]interface{}{
				"format":                     asrConfig.Format,
				"sample_rate":                asrConfig.SampleRate,
				"disfluency_removal_enabled": false,
				"language_hints":             asrConfig.LanguageHints,
			},
			"input": map[string]interface{}{},
		},
	}
	if err := aliWS.WriteJSON(runTask); err != nil {
		return err
	}

	// 等待任务启动
	var started bool
	for !started {
		_, msg, err := aliWS.ReadMessage()
		if err != nil {
			return err
		}
		var response map[string]interface{}
		if err := json.Unmarshal(msg, &response); err != nil {
			continue
		}
		header := response["header"].(map[string]interface{})
		if header["event"] == "task-started" {
			started = true
		}
	}

	// 使用WaitGroup管理协程
	var wg sync.WaitGroup
	wg.Add(2)

	// 协程：将客户端音频转发到阿里云（二进制）
	go func() {
		defer wg.Done()
		for {
			mt, data, err := clientWS.ReadMessage()
			if err != nil {
				break
			}
			if mt == websocket.BinaryMessage {
				if err := aliWS.WriteMessage(websocket.BinaryMessage, data); err != nil {
					break
				}
			}
		}
		// 任务结束时发送finish-task
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
		aliWS.WriteJSON(finishTask)
	}()

	// 协程：从阿里云读取结果并将文本发送给AI服务
	go func() {
		defer wg.Done()
		for {
			_, msg, err := aliWS.ReadMessage()
			if err != nil {
				break
			}
			var response map[string]interface{}
			if err := json.Unmarshal(msg, &response); err != nil {
				continue
			}
			header := response["header"].(map[string]interface{})
			event := header["event"].(string)
			if event == "result-generated" {
				payload := response["payload"].(map[string]interface{})
				output := payload["output"].(map[string]interface{})
				sentence := output["sentence"].(map[string]interface{})
				text := sentence["text"].(string)
				aiResponse, err := client.GenerateResponse(ctx, text, "")
				if err != nil {
					log.Printf("AI服务错误: %v", err)
					continue
				}
				// 将AI响应发送给客户端
				clientWS.WriteJSON(map[string]string{"type": "asr_text", "text": aiResponse})
			} else if event == "task-finished" || event == "task-failed" {
				break
			}
		}
	}()

	wg.Wait()
	return nil
}
