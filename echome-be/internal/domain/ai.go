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

	// VoiceClone 执行语音克隆操作
	// @param ctx 上下文
	// @param config 语音克隆配置
	// @return string 克隆的声音ID
	VoiceClone(ctx context.Context, config *VoiceCloneConfig) (*string, error)
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
	Model          string  // qwen3-tts-flash-realtime / cosyvoice-v2
	Voice          string  // 克隆音色的voice_id/	非克隆时选择模型自带角色名
	Format string  // pcm / mp3
	Mode           string  // server_commit / commit
	Lang           string  // 语言类型，如"zh"、"en"等
}

// VoiceCloneConfig 定义语音克隆配置参数
type VoiceCloneConfig struct {
	// 克隆目标模型
	TargetModel string `json:"target_model"`
	// 音频URL，必须是公网可访问的URL
	AudioURL string `json:"audio_url"`
	// 音频文件名
	AudioFilename string `json:"audio_filename,omitempty"`
	// 克隆的声音名称
	VoiceName string `json:"voice_name"`
	// 克隆的声音描述
	VoiceDescription string `json:"voice_description,omitempty"`
	// 语言类型，如"zh"、"en"等
	LanguageType string `json:"language_type"`
}
