package config

type Aliyun struct {
	APIKey   string           `mapstructure:"api_key"`
	Endpoint string           `mapstructure:"endpoint"`
	Region   string           `mapstructure:"region"`
	ASR      ASRServiceConfig `mapstructure:"asr"`
	TTS      TTSServiceConfig `mapstructure:"tts"`
	LLM      LLMServiceConfig `mapstructure:"llm"`
}

// ASRServiceConfig defines ASR service configuration
type ASRServiceConfig struct {
	Model         string   `mapstructure:"model"`
	SampleRate    int      `mapstructure:"sample_rate"`
	Format        string   `mapstructure:"format"`
	LanguageHints []string `mapstructure:"language_hints"`
}

// TTSServiceConfig defines TTS service configuration
type TTSServiceConfig struct {
	Model          string `mapstructure:"model"`
	DefaultVoice   string `mapstructure:"default_voice"`
	SampleRate     int    `mapstructure:"sample_rate"`
	ResponseFormat string `mapstructure:"response_format"`
}

// LLMServiceConfig defines LLM service configuration
type LLMServiceConfig struct {
	Model       string  `mapstructure:"model"`
	Temperature float32 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

// 阿里云相关的常量和默认值
const (
	// DefaultALBLEndpoint 阿里云百炼服务的默认端点
	DefaultALBLEndpoint = "https://dashscope.aliyuncs.com"

	// ALBLServiceType 阿里云百炼服务类型
	ALBLServiceType = "alibailian"
)
