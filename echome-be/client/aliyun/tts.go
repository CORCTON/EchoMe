package aliyun

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
	"golang.org/x/sync/errgroup"
)

// DefaultTTSConfig 返回默认的阿里云百炼TTS配置
func DefaultTTSConfig() domain.TTSConfig {
	return domain.TTSConfig{
		Model:          "qwen3-tts-flash-realtime",
		Voice:          "Katerina",
		ResponseFormat: "pcm",
		SampleRate:     24000,
		Mode:           "server_commit",
	}
}

// HandleTTS 通过阿里云百炼WebSocket处理TTS
func (client *AliClient) HandleTTS(ctx context.Context, clientWS *websocket.Conn, config domain.TTSConfig) error {
	aliWS, err := connectToAliyunTTS(client.apiKey, client.endPoint, config.Model)
	if err != nil {
		return err
	}
	defer aliWS.Close()

	// 更新会话配置
	if err := updateTTSSession(aliWS, config); err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	// 从客户端读取文本并发送到阿里云
	g.Go(func() error {
		return handleClientToAliyun(ctx, clientWS, aliWS, config.Mode)
	})

	// 从阿里云读取音频并发送到客户端
	g.Go(func() error {
		return handleAliyunToClient(ctx, aliWS, clientWS)
	})

	// 添加心跳保持连接
	g.Go(func() error {
		return keepAlive(ctx, aliWS)
	})

	return g.Wait()
}

// connectToAliyunTTS 连接到阿里云百炼TTS WebSocket
func connectToAliyunTTS(apiKey, endpoint, model string) (*websocket.Conn, error) {
	// 构建阿里云百炼TTS WebSocket URL，使用配置中的模型
	baseURL := strings.Replace(endpoint, "https://", "wss://", 1)
	url := fmt.Sprintf("%s/api-ws/v1/realtime?model=%s", baseURL, model)

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+apiKey)

	ws, resp, err := dialer.Dial(url, headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("连接阿里云百炼TTS失败: %v, 状态码: %d", err, resp.StatusCode)
		}
		return nil, fmt.Errorf("连接阿里云百炼TTS失败: %v", err)
	}

	// 设置WebSocket参数
	ws.SetPongHandler(func(appData string) error {
		return nil
	})

	return ws, nil
}

// keepAlive 保持WebSocket连接活跃
func keepAlive(ctx context.Context, ws *websocket.Conn) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return err
			}
		}
	}
}

// updateTTSSession 更新TTS会话配置
func updateTTSSession(aliWS *websocket.Conn, config domain.TTSConfig) error {
	eventID := fmt.Sprintf("event_%d", time.Now().UnixMilli())

	sessionUpdate := map[string]interface{}{
		"type":     "session.update",
		"event_id": eventID,
		"session": map[string]interface{}{
			"mode":            config.Mode,
			"voice":           config.Voice,
			"language_type":   "Auto",
			"response_format": config.ResponseFormat,
			"sample_rate":     config.SampleRate,
		},
	}

	if err := aliWS.WriteJSON(sessionUpdate); err != nil {
		return fmt.Errorf("发送session配置失败: %w", err)
	}

	// 等待session.updated或session.created响应
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-timeout.C:
			return fmt.Errorf("等待session配置响应超时")
		default:
			aliWS.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, msg, err := aliWS.ReadMessage()
			if err != nil {
				return fmt.Errorf("读取session配置响应失败: %w", err)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(msg, &response); err != nil {
				continue
			}

			eventType, ok := response["type"].(string)
			if !ok {
				continue
			}

			switch eventType {
			case "session.updated", "session.created":
				if session, ok := response["session"].(map[string]interface{}); ok {
					if _, ok := session["id"].(string); ok {
						aliWS.SetReadDeadline(time.Time{})
						return nil
					}
				}
			case "error":
				if errorMsg, ok := response["error"].(map[string]interface{}); ok {
					return fmt.Errorf("session配置错误: %v", errorMsg)
				}
				return fmt.Errorf("session配置失败: %v", response)
			}
		}
	}
}

// sendEvent 发送事件到阿里云
func sendEvent(aliWS *websocket.Conn, eventType string, data map[string]interface{}) error {
	event := map[string]any{
		"type":     eventType,
		"event_id": fmt.Sprintf("event_%d", time.Now().UnixMilli()),
	}

	// 合并额外数据
	for k, v := range data {
		event[k] = v
	}

	return aliWS.WriteJSON(event)
}

// handleClientToAliyun 处理从客户端到阿里云百炼的文本传输
func handleClientToAliyun(ctx context.Context, clientWS, aliWS *websocket.Conn, mode string) error {
	defer func() {
		// 发送会话结束信号
		sendEvent(aliWS, "session.finish", nil)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			clientWS.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, data, err := clientWS.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					return nil
				}
				return nil
			}

			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			// 处理不同类型的消息
			if msgType, ok := msg["type"].(string); ok {
				switch msgType {
				case "text":
					if text, ok := msg["text"].(string); ok && text != "" {
						if err := sendTextToAliyun(aliWS, text, mode); err != nil {
							return fmt.Errorf("发送文本到阿里云百炼失败: %w", err)
						}
					}
				case "finish":
					return nil
				}
			} else if text, ok := msg["text"].(string); ok && text != "" {
				// 兼容旧格式
				if err := sendTextToAliyun(aliWS, text, mode); err != nil {
					return fmt.Errorf("发送文本到阿里云百炼失败: %w", err)
				}
			}
		}
	}
}

// sendTextToAliyun 发送文本到阿里云百炼TTS
func sendTextToAliyun(aliWS *websocket.Conn, text, mode string) error {
	// 发送文本追加消息
	if err := sendEvent(aliWS, "input_text_buffer.append", map[string]interface{}{
		"text": text,
	}); err != nil {
		return fmt.Errorf("发送文本追加消息失败: %w", err)
	}

	// 根据模式决定是否立即提交
	if mode == "commit" || mode == "server_commit" {
		if err := sendEvent(aliWS, "input_text_buffer.commit", nil); err != nil {
			return fmt.Errorf("发送提交消息失败: %w", err)
		}
	}

	return nil
}

// handleAliyunToClient 处理从阿里云百炼到客户端的音频传输
func handleAliyunToClient(ctx context.Context, aliWS, clientWS *websocket.Conn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			aliWS.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, msg, err := aliWS.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					return nil
				}
				return nil
			}

			var response map[string]interface{}
			if err := json.Unmarshal(msg, &response); err != nil {
				continue
			}

			eventType, ok := response["type"].(string)
			if !ok {
				continue
			}

			switch eventType {
			case "response.audio.delta":
				// 处理音频数据流
				var audioData string
				if delta, ok := response["delta"].(string); ok && delta != "" {
					audioData = delta
				} else if audio, ok := response["audio"].(string); ok && audio != "" {
					audioData = audio
				} else if data, ok := response["data"].(string); ok && data != "" {
					audioData = data
				}

				if audioData != "" {
					audioBytes, err := base64.StdEncoding.DecodeString(audioData)
					if err != nil {
						continue
					}

					// 发送到客户端
					if err := clientWS.WriteMessage(websocket.BinaryMessage, audioBytes); err != nil {
						return fmt.Errorf("向客户端发送音频失败: %w", err)
					}
				}

			case "response.audio.done":
				// 音频生成完成
				continue

			case "response.done":
				// 响应完成
				continue

			case "session.finished":
				// 会话结束
				return nil

			case "error":
				// 处理错误
				if errorInfo, ok := response["error"].(map[string]interface{}); ok {
					return fmt.Errorf("阿里云百炼TTS错误: %v", errorInfo)
				}
				return fmt.Errorf("阿里云百炼TTS未知错误: %v", response)

			default:
				// 忽略其他类型的消息
				continue
			}
		}
	}
}
