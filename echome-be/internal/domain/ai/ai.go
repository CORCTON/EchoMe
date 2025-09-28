package ai

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
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters"`
}

// ToolCallResponse 工具调用响应结构
type ToolCallResponse struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// DashScopeChatRequest 阿里云DashScope请求结构
type DashScopeChatRequest struct {
	Model        string           `json:"model"`
	Messages     []map[string]any `json:"messages"`
	Stream       bool             `json:"stream"`
	EnableSearch bool             `json:"enable_search,omitempty"`
	Tools        []map[string]any `json:"tools,omitempty"`
}

// DashScopeStreamChunk DashScope流式响应块结构
type DashScopeStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string     `json:"content,omitempty"`
			Role      string     `json:"role,omitempty"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
}
