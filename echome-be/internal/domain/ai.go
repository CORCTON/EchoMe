package domain

import (
	"context"

	"github.com/gorilla/websocket"
)

// AIService 定义AI服务接口
// @Description AI服务接口，提供生成响应的能力
type AIService interface {
	// HandleASR 处理ASR请求
	HandleASR(ctx context.Context, clientWS *websocket.Conn) error

	// HandleTTS 处理TTS请求（从WebSocket读取文本）
	HandleTTS(ctx context.Context, clientWS *websocket.Conn) error

	// TextToSpeech 直接将文本转换为语音并发送到WebSocket
	// @param ctx 上下文
	// @param text 需要转换的文本
	// @param writer WebSocket写入器
	TextToSpeech(ctx context.Context, text string, writer WSWriter) error

	// GenerateResponse 生成AI响应
	// @param ctx 上下文
	// @param userInput 用户输入
	// @param characterContext 角色上下文
	// @param conversationHistory 对话历史，格式为[{"role": "user/assistant", "content": "内容"}, ...]
	GenerateResponse(ctx context.Context, userInput string, characterContext string, conversationHistory []map[string]string) (string, error)

	// GenerateStreamResponse 生成AI流式响应
	// @param ctx 上下文
	// @param userInput 用户输入
	// @param characterContext 角色上下文
	// @param conversationHistory 对话历史，格式为[{"role": "user/assistant", "content": "内容"}, ...]
	// @param onChunk 处理文本块的回调函数
	GenerateStreamResponse(ctx context.Context, userInput string, characterContext string, conversationHistory []map[string]string, onChunk func(string) error) error
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
