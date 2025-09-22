package domain

import (
	"context"

	"github.com/gorilla/websocket"
)

// AIService 定义AI服务接口
// @Description AI服务接口，提供生成响应的能力
type AIService interface {
	// HandleASR 处理ASR请求
	HandleASR(ctx context.Context, clientWS *websocket.Conn, config ASRConfig) error

	// HandleTTS 处理TTS请求
	HandleTTS(ctx context.Context, clientWS *websocket.Conn, config TTSConfig) error

	// GenerateResponse 生成AI响应
	GenerateResponse(ctx context.Context, userInput string, characterContext string) (string, error)
}

// ASRConfig 定义ASR配置参数
type ASRConfig struct {
	Model         string   `json:"model"`
	Format        string   `json:"format"`
	SampleRate    int      `json:"sample_rate"`
	LanguageHints []string `json:"language_hints,omitempty"`
}

// TTSConfig 定义TTS配置参数
type TTSConfig struct {
	Model          string `json:"model"`
	Voice          string `json:"voice"`
	ResponseFormat string `json:"response_format"`
	SampleRate     int    `json:"sample_rate"`
	Mode           string `json:"mode"` // "server_commit" or "commit"
}
