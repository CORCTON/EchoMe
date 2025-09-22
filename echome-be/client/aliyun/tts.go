package aliyun

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
)

// DefaultTTSConfig returns default config for Qwen-TTS
func DefaultTTSConfig() domain.TTSConfig {
	return domain.TTSConfig{
		Model:          "qwen-tts-realtime",
		Voice:          "Cherry", // Example voice, change as needed
		ResponseFormat: "pcm",
		SampleRate:     24000,
		Mode:           "server_commit",
	}
}

// HandleTTS processes TTS via Aliyun WebSocket
func (client *AliClient) HandleTTS(ctx context.Context, clientWS *websocket.Conn, config domain.TTSConfig) error {
	// Connect to Aliyun TTS WebSocket
	url := fmt.Sprintf("wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=%s", config.Model)
	dialer := websocket.Dialer{}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+client.apiKey)
	aliWS, _, err := dialer.Dial(url, headers)
	if err != nil {
		return err
	}
	defer aliWS.Close()

	// Send session.update to configure
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
		return err
	}

	// Wait for session.updated
	var sessionID string
	for sessionID == "" {
		_, msg, err := aliWS.ReadMessage()
		if err != nil {
			return err
		}
		var response map[string]interface{}
		if err := json.Unmarshal(msg, &response); err != nil {
			continue
		}
		eventType := response["type"].(string)
		if eventType == "session.updated" {
			session := response["session"].(map[string]interface{})
			sessionID = session["id"].(string)
		}
	}

	// Use wait group for goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine to read text from client and send to Aliyun
	go func() {
		defer wg.Done()
		for {
			_, data, err := clientWS.ReadMessage()
			if err != nil {
				break
			}
			var msg map[string]interface{}
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			if text, ok := msg["text"].(string); ok {
				eventID := fmt.Sprintf("event_%d", time.Now().UnixMilli())
				appendMsg := map[string]interface{}{
					"type":     "input_text_buffer.append",
					"event_id": eventID,
					"text":     text,
				}
				aliWS.WriteJSON(appendMsg)

				// If mode is commit, send commit
				if config.Mode == "commit" {
					commitMsg := map[string]interface{}{
						"type":     "input_text_buffer.commit",
						"event_id": fmt.Sprintf("event_%d", time.Now().UnixMilli()),
					}
					aliWS.WriteJSON(commitMsg)
				}
			}
		}
		// Send session.finish when done
		finishMsg := map[string]interface{}{
			"type":     "session.finish",
			"event_id": fmt.Sprintf("event_%d", time.Now().UnixMilli()),
		}
		aliWS.WriteJSON(finishMsg)
	}()

	// Goroutine to read audio from Aliyun and send to client
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
			eventType := response["type"].(string)
			if eventType == "response.audio.delta" {
				audioDelta := response["audio"].(string)
				// Decode base64 to bytes
				audioBytes, err := base64.StdEncoding.DecodeString(audioDelta)
				if err != nil {
					continue
				}
				// Send binary audio to client
				clientWS.WriteMessage(websocket.BinaryMessage, audioBytes)
			} else if eventType == "session.finished" {
				break
			}
		}
	}()

	wg.Wait()
	return nil
}
