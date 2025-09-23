package aliyun

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
	"golang.org/x/sync/errgroup"
)

// DefaultTTSConfig 返回默认的 Qwen-TTS 配置
func DefaultTTSConfig() domain.TTSConfig {
	return domain.TTSConfig{
		Model:          "qwen-tts-realtime",
		Voice:          "Cherry",
		ResponseFormat: "pcm",
		SampleRate:     24000,
		Mode:           "server_commit",
	}
}

// HandleTTS 通过 Aliyun WebSocket 处理 TTS
func (client *AliClient) HandleTTS(ctx context.Context, clientWS *websocket.Conn, config domain.TTSConfig) error {
	aliWS, err := connectToAliyunTTS(client.apiKey, config.Model)
	if err != nil {
		return err
	}
	defer aliWS.Close()

	sessionID, err := configureTTSSession(aliWS, config)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	// 从客户端读取文本并发送到 Aliyun
	g.Go(func() error {
		return handleClientToAliyun(ctx, clientWS, aliWS, config.Mode, sessionID)
	})

	// 从 Aliyun 读取音频并发送到客户端
	g.Go(func() error {
		return handleAliyunToClient(ctx, aliWS, clientWS)
	})

	return g.Wait()
}

// connectToAliyunTTS 连接到 Aliyun TTS WebSocket
func connectToAliyunTTS(apiKey, model string) (*websocket.Conn, error) {
	url := fmt.Sprintf("wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=%s", model)
	dialer := websocket.Dialer{}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+apiKey)
	ws, _, err := dialer.Dial(url, headers)
	return ws, err
}

// configureTTSSession 配置 TTS 会话并返回 session ID
func configureTTSSession(aliWS *websocket.Conn, config domain.TTSConfig) (string, error) {
	eventID := fmt.Sprintf("event_%d", time.Now().UnixMilli())
	sessionUpdate := map[string]interface{}{
		"type":     "session.update",
		"event_id": eventID,
		"session": map[string]interface{}{
			"mode":            config.Mode,
			"voice":           config.Voice,
			"response_format": config.ResponseFormat,
			"sample_rate":     config.SampleRate,
		},
	}
	if err := aliWS.WriteJSON(sessionUpdate); err != nil {
		return "", err
	}

	var sessionID string
	for sessionID == "" {
		_, msg, err := aliWS.ReadMessage()
		if err != nil {
			return "", err
		}
		var response map[string]interface{}
		if err := json.Unmarshal(msg, &response); err != nil {
			continue
		}
		if eventType, ok := response["type"].(string); ok && eventType == "session.updated" {
			if session, ok := response["session"].(map[string]interface{}); ok {
				if id, ok := session["id"].(string); ok {
					sessionID = id
				}
			}
		}
	}
	return sessionID, nil
}

// handleClientToAliyun 处理从客户端到 Aliyun 的文本传输
func handleClientToAliyun(ctx context.Context, clientWS, aliWS *websocket.Conn, mode, sessionID string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, data, err := clientWS.ReadMessage()
			if err != nil {
				return nil // 客户端断开连接
			}
			var msg map[string]interface{}
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			text, ok := msg["text"].(string)
			if !ok {
				continue
			}
			eventID := fmt.Sprintf("event_%d", time.Now().UnixMilli())
			appendMsg := map[string]interface{}{
				"type":      "input_text_buffer.append",
				"event_id":  eventID,
				"session_id": sessionID, // 添加 sessionID
				"text":      text,
			}
			if err := aliWS.WriteJSON(appendMsg); err != nil {
				return fmt.Errorf("向阿里云发送文本失败: %w", err)
			}

			if mode == "commit" {
				commitMsg := map[string]interface{}{
					"type":      "input_text_buffer.commit",
					"event_id":  fmt.Sprintf("event_%d", time.Now().UnixMilli()),
					"session_id": sessionID, // 添加 sessionID
				}
				if err := aliWS.WriteJSON(commitMsg); err != nil {
					return fmt.Errorf("向阿里云发送提交消息失败: %w", err)
				}
			}
		}
	}
}

// handleAliyunToClient 处理从 Aliyun 到客户端的音频传输
func handleAliyunToClient(ctx context.Context, aliWS, clientWS *websocket.Conn) error {
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
			eventType, ok := response["type"].(string)
			if !ok {
				continue
			}
			if eventType == "response.audio.delta" {
				audioDelta, ok := response["audio"].(string)
				if !ok {
					continue
				}
				audioBytes, err := base64.StdEncoding.DecodeString(audioDelta)
				if err != nil {
					continue
				}
				if err := clientWS.WriteMessage(websocket.BinaryMessage, audioBytes); err != nil {
					return fmt.Errorf("向客户端发送音频失败: %w", err)
				}
			} else if eventType == "session.finished" {
				return nil
			}
		}
	}
}