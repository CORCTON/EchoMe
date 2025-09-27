package domain

import (
	"context"
)

// AIService AI服务
type AIService interface {
	// HandleASR 处理ASR请求
	HandleASR(ctx context.Context, clientWS WebSocketConn) error

	// HandleTTS 处理TTS请求（从WebSocket读取文本）
	// @param ctx 上下文
	// @param clientWS WebSocket连接
	// @param text 要转换为语音的文本
	// @param config TTS配置参数
	HandleTTS(ctx context.Context, clientWS WebSocketConn, text string, config TTSConfig) error

	// HandleCosyVoiceTTS handles the TTS WebSocket connection using CosyVoice streaming API.
	HandleCosyVoiceTTS(ctx context.Context, clientWS WebSocketConn, textStream <-chan string, config TTSConfig) error
	// GenerateResponse LLM响应
	// @param ctx 上下文
	// @param userInput 用户输入
	// @param characterContext 角色上下文
	// @param conversationHistory 对话历史，格式为[{"role": "user/assistant", "content": "内容"}, ...]
	// @param onChunk 处理文本块的回调函数
	GenerateResponse(ctx context.Context, msg DashScopeChatRequest, onChunk func(string) error) error



	// VoiceClone 执行语音克隆操作
	// @param ctx 上下文
	// @param url 音频URL
	// @return string 克隆的声音ID
	VoiceClone(ctx context.Context, url string) (*string, error)

	// GetVoiceStatus 查询音色状态
	// @param ctx 上下文
	// @param voiceID 音色ID
	// @return bool 音色是否可用
	// @return error 错误信息
	GetVoiceStatus(ctx context.Context, voiceID string) (bool, error)
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
	Model  string // qwen3-tts-flash-realtime / cosyvoice-v2
	Voice  string // 克隆音色的voice_id/	非克隆时选择模型自带角色名
	Format string // pcm / mp3
	Mode   string // server_commit / commit
	Lang   string // 语言类型，如"zh"、"en"等
}

// ToolCall 工具调用结构
type ToolCall struct {
	Name     string                 `json:"name"`
	Parameters map[string]any      `json:"parameters"`
}

// ToolCallResponse 工具调用响应结构
type ToolCallResponse struct {
	Name     string                 `json:"name"`
	Content  string                 `json:"content"`
}

// DashScopeChatRequest 阿里云DashScope请求结构
type DashScopeChatRequest struct {
	Model    string              `json:"model"`
	Messages []map[string]any `json:"messages"`
	Stream   bool                `json:"stream"`
	EnableSearch bool                `json:"enable_search,omitempty"`
	Tools    []map[string]any  `json:"tools,omitempty"`
}

// DashScopeStreamChunk DashScope流式响应块结构
type DashScopeStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content,omitempty"`
			Role    string `json:"role,omitempty"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
}
