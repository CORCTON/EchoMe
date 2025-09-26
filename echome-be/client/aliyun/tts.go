package aliyun

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
	"golang.org/x/sync/errgroup"
)

// DefaultTTSConfig 提供默认 TTS 配置
func DefaultTTSConfig() domain.TTSConfig {
	return domain.TTSConfig{
		Model: "paraformer-realtime-v2",
		Voice: "Jennifer",
		Format: "pcm",
	}
}

// HandleTTS 处理 TTS 请求，直接使用传入的文本
func (client *AliClient) HandleTTS(ctx context.Context, clientWS domain.WebSocketConn, text string, config domain.TTSConfig) error {
	textCh := make(chan string, 1)
	textCh <- text
	close(textCh)
	return client.HandleCosyVoiceTTS(ctx, clientWS, textCh, config)
}

func (client *AliClient) HandleCosyVoiceTTS(ctx context.Context, clientWS domain.WebSocketConn, textStream <-chan string, config domain.TTSConfig) error {
	aliWS, err := connectToAliyunTTS(client.apiKey)
	if err != nil {
		return fmt.Errorf("连接阿里云 TTS 失败: %w", err)
	}
	defer aliWS.Close()

	taskID := uuid.NewString()
	taskStarted := make(chan struct{})
	g, ctx := errgroup.WithContext(ctx)

	// 1. 从阿里云读取消息并转发
	g.Go(func() error {
		return handleAliyunToClient(ctx, aliWS, clientWS, taskStarted)
	})

	// 2. 发送指令到阿里云
	g.Go(func() error {
		// 等待 task-started 事件
		select {
		case <-taskStarted:
			// 收到事件，继续执行
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
			return fmt.Errorf("等待 task-started 超时")
		}

		// 循环发送文本
		for text := range textStream {
			if err := sendContinueTask(aliWS, taskID, text); err != nil {
				return fmt.Errorf("发送 continue-task 失败: %w", err)
			}
		}

		// 所有文本发送完毕，发送 finish-task
		if err := sendFinishTask(aliWS, taskID); err != nil {
			return fmt.Errorf("发送 finish-task 失败: %w", err)
		}
		return nil
	})

	// 3. 发送 run-task 指令
	if err := sendRunTask(aliWS, taskID, config); err != nil {
		return fmt.Errorf("发送 run-task 失败: %w", err)
	}

	return g.Wait()
}

// connectToAliyunTTS 建立 WebSocket 连接
func connectToAliyunTTS(apiKey string) (*websocket.Conn, error) {
	url := "wss://dashscope.aliyuncs.com/api-ws/v1/inference"

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+apiKey)
	headers.Add("X-DashScope-DataInspection", "enable")

	ws, resp, err := dialer.Dial(url, headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("连接失败: %v, 状态码: %d", err, resp.StatusCode)
		}
		return nil, fmt.Errorf("连接失败: %v", err)
	}

	return ws, nil
}

// sendRunTask 发送开启任务指令
func sendRunTask(ws *websocket.Conn, taskID string, config domain.TTSConfig) error {
	cmd := map[string]interface{}{
		"header": map[string]interface{}{
			"action":    "run-task",
			"task_id":   taskID,
			"streaming": "duplex",
		},
		"payload": map[string]interface{}{
			"task_group": "audio",
			"task":       "tts",
			"function":   "SpeechSynthesizer",
			"model":      config.Model,
			"parameters": map[string]interface{}{
				"text_type":   "PlainText",
				"voice":       config.Voice,
				"format":      config.Format,
				"sample_rate": 22050,
			},
			"input": map[string]any{},
		},
	}
	return ws.WriteJSON(cmd)
}

// sendContinueTask 发送待合成文本
func sendContinueTask(ws *websocket.Conn, taskID, text string) error {
	cmd := map[string]any{
		"header": map[string]any{
			"action":    "continue-task",
			"task_id":   taskID,
			"streaming": "duplex",
		},
		"payload": map[string]any{
			"input": map[string]any{
				"text": text,
			},
		},
	}
	return ws.WriteJSON(cmd)
}

// sendFinishTask 发送结束任务指令
func sendFinishTask(ws *websocket.Conn, taskID string) error {
	cmd := map[string]interface{}{
		"header": map[string]interface{}{
			"action":    "finish-task",
			"task_id":   taskID,
			"streaming": "duplex",
		},
		"payload": map[string]interface{}{
			"input": map[string]interface{}{},
		},
	}
	return ws.WriteJSON(cmd)
}

// handleAliyunToClient 从阿里云读取消息并转发
func handleAliyunToClient(ctx context.Context, aliWS *websocket.Conn, writer domain.WebSocketConn, taskStarted chan<- struct{}) error {
	defer close(taskStarted)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msgType, msg, err := aliWS.ReadMessage()
			if err != nil {
				// 正常关闭或上下文取消时，返回 nil
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) || strings.Contains(err.Error(), "context canceled") {
					return nil
				}
				return fmt.Errorf("读取阿里云消息失败: %w", err)
			}

			if msgType == websocket.BinaryMessage {
				if err := writer.WriteMessage(websocket.BinaryMessage, msg); err != nil {
					return fmt.Errorf("转发音频失败: %w", err)
				}
				continue
			}

			if msgType != websocket.TextMessage {
				continue
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(msg, &resp); err != nil {
			zap.L().Warn("解析阿里云 JSON 失败", zap.Error(err))
			continue
		}

			header, ok := resp["header"].(map[string]interface{})
			if !ok {
				continue
			}

			event, ok := header["event"].(string)
			if !ok {
				continue
			}

			switch event {
			case "task-started":
				select {
				case taskStarted <- struct{}{}:
				default:
				}
			case "task-finished":
				zap.L().Info("阿里云 TTS 任务完成")
				return nil
			case "task-failed":
				errMsg, _ := header["error_message"].(string)
				return fmt.Errorf("阿里云 TTS 任务失败: %s", errMsg)
			case "result-generated":
				// 忽略
			}
		}
	}
}
