package aliyun

import (
	"context"
	"crypto/tls"
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

// 通用模型 TTS 默认配置
func DefaultTTSConfig() domain.TTSConfig {
	return domain.TTSConfig{
		Model:          "qwen3-tts-flash-realtime",
		Voice:          "Katerina",
		Mode:           "server_commit",
		Format:         "pcm",
	}
}

// HandleTTS 从 WebSocket 客户端读取文本并转成语音
func (client *AliClient) HandleTTS(ctx context.Context, clientWS domain.WebSocketConn, config domain.TTSConfig) error {
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

	// 从客户端读取文本 -> 发送到阿里云
	g.Go(func() error {
		return handleClientToAliyun(ctx, clientWS, aliWS, config)
	})

	// 从阿里云读取音频 -> 写给客户端
	g.Go(func() error {
		return handleAliyunToClient(ctx, aliWS, clientWS)
	})

	// 心跳
	g.Go(func() error {
		return keepAlive(ctx, aliWS)
	})

	return g.Wait()
}

// TextToSpeech 直接输入文本 -> 返回音频
func (client *AliClient) TextToSpeech(ctx context.Context, text string, writer domain.WebSocketConn, config domain.TTSConfig) error {
	aliWS, err := connectToAliyunTTS(client.apiKey, client.endPoint, config.Model)
	if err != nil {
		return fmt.Errorf("连接阿里云百炼TTS失败: %w", err)
	}
	defer aliWS.Close()

	if err := updateTTSSession(aliWS, config); err != nil {
		return fmt.Errorf("更新TTS会话配置失败: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	// 发送文本
	g.Go(func() error {
		if err := sendTextToAliyun(aliWS, text, config); err != nil {
			return fmt.Errorf("发送文本失败: %w", err)
		}
		time.Sleep(100 * time.Millisecond)
		return sendEvent(aliWS, "session.finish", nil)
	})

	// 接收音频
	g.Go(func() error {
		return handleAliyunToClient(ctx, aliWS, writer)
	})

	// 心跳
	g.Go(func() error {
		return keepAlive(ctx, aliWS)
	})

	return g.Wait()
}

// connectToAliyunTTS 建立 WebSocket 连接
func connectToAliyunTTS(apiKey, endpoint, model string) (*websocket.Conn, error) {
	baseURL := strings.Replace(endpoint, "https://", "wss://", 1)
	url := fmt.Sprintf("%s/api-ws/v1/realtime?model=%s", baseURL, model)

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+apiKey)

	ws, resp, err := dialer.Dial(url, headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("连接失败: %v, 状态码: %d", err, resp.StatusCode)
		}
		return nil, fmt.Errorf("连接失败: %v", err)
	}

	ws.SetPongHandler(func(string) error { return nil })
	return ws, nil
}

// keepAlive 定时心跳
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

// updateTTSSession 初始化配置
func updateTTSSession(aliWS *websocket.Conn, config domain.TTSConfig) error {
	// 根据模型选择不同处理方式
	if config.Model == "cosyvoice-v2" {
		sessionUpdate := map[string]interface{}{
			"header": map[string]interface{}{
				"action":   "run-task",
				"task_id":  fmt.Sprintf("task_%d", time.Now().UnixNano()),
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
				},
				"input": map[string]interface{}{},
			},
		}
		return aliWS.WriteJSON(sessionUpdate)
	}

	// qwen3 默认配置
	sessionUpdate := map[string]interface{}{
		"type": "session.update",
		"session": map[string]any{
			"mode":            config.Mode,
			"voice":           config.Voice,
			"language_type":   "Auto",
			"language_hints":  config.Lang,
			"format": config.Format,
		},
	}
	return aliWS.WriteJSON(sessionUpdate)
}

// sendEvent 统一发事件
func sendEvent(aliWS *websocket.Conn, eventType string, data map[string]interface{}) error {
	event := map[string]any{
		"type":     eventType,
		"event_id": fmt.Sprintf("event_%d", time.Now().UnixMilli()),
	}
	for k, v := range data {
		event[k] = v
	}
	return aliWS.WriteJSON(event)
}

// handleClientToAliyun 文本转发
func handleClientToAliyun(ctx context.Context, clientWS domain.WebSocketConn, aliWS *websocket.Conn, config domain.TTSConfig) error {
	defer sendEvent(aliWS, "session.finish", nil)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			clientWS.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, data, err := clientWS.ReadMessage()
			if err != nil {
				return nil
			}

			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			if text, ok := msg["text"].(string); ok && text != "" {
				if err := sendTextToAliyun(aliWS, text, config); err != nil {
					return err
				}
			}
		}
	}
}

// sendTextToAliyun 发文本
func sendTextToAliyun(aliWS *websocket.Conn, text string, config domain.TTSConfig) error {
	// 如果是克隆音色
	if config.Model == "cosyvoice-v2" {
		event := map[string]interface{}{
			"header": map[string]interface{}{
				"action":   "run-task",
				"task_id":  fmt.Sprintf("task_%d", time.Now().UnixNano()),
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
				},
				"input": map[string]interface{}{
					"text": text,
				},
			},
		}
		return aliWS.WriteJSON(event)
	}

	// qwen3 默认
	if err := sendEvent(aliWS, "input_text_buffer.append", map[string]interface{}{
		"text": text,
	}); err != nil {
		return err
	}
	if config.Mode == "commit" || config.Mode == "server_commit" {
		return sendEvent(aliWS, "input_text_buffer.commit", nil)
	}
	return nil
}

// handleAliyunToClient 音频转发
func handleAliyunToClient(ctx context.Context, aliWS *websocket.Conn, writer domain.WebSocketConn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, msg, err := aliWS.ReadMessage()
			if err != nil {
				return nil
			}
			var resp map[string]interface{}
			if err := json.Unmarshal(msg, &resp); err != nil {
				continue
			}

			switch resp["type"] {
			case "response.audio.delta", "response.audio":
				var audioData string
				if delta, ok := resp["delta"].(string); ok && delta != "" {
					audioData = delta
				} else if audio, ok := resp["audio"].(string); ok && audio != "" {
					audioData = audio
				} else if data, ok := resp["data"].(string); ok && data != "" {
					audioData = data
				}
				if audioData != "" {
					audioBytes, err := base64.StdEncoding.DecodeString(audioData)
					if err != nil {
						continue
					}
					if err := writer.WriteMessage(websocket.BinaryMessage, audioBytes); err != nil {
						return err
					}
				}
			case "session.finished":
				return nil
			case "error":
				return fmt.Errorf("TTS错误: %v", resp)
			}
		}
	}
}
