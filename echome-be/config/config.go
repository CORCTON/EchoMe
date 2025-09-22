package config

import (
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// MustLoad 是一个泛型配置加载函数，发生错误时会panic
func MustLoad[T any](path string) (*koanf.Koanf, *T) {
	k := koanf.New(".")

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		panic(err)
	}

	var config T
	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "mapstructure"}); err != nil {
		panic(err)
	}
	return k, &config
}

func Load(path string) *Config {
	_, c := MustLoad[Config](path)
	return c
}

// Config holds all application configuration
type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	OpenAI struct {
		APIKey string `mapstructure:"api_key"`
	} `mapstructure:"openai"`
	ALBL struct {
		APIKey    string `mapstructure:"api_key"`
		APISecret string `mapstructure:"api_secret"`
		Endpoint  string `mapstructure:"endpoint"`
	} `mapstructure:"albl"`
	WebRTC struct {
		STUNServer string `mapstructure:"stun_server"`
	} `mapstructure:"webrtc"`
	AI struct {
		ServiceType string `mapstructure:"service_type"`
		Timeout     int    `mapstructure:"timeout"`
		MaxRetries  int    `mapstructure:"max_retries"`
	} `mapstructure:"ai"`
	Aliyun struct {
		APIKey   string           `mapstructure:"api_key"`
		Endpoint string           `mapstructure:"endpoint"`
		Region   string           `mapstructure:"region"`
		ASR      ASRServiceConfig `mapstructure:"asr"`
		TTS      TTSServiceConfig `mapstructure:"tts"`
		LLM      LLMServiceConfig `mapstructure:"llm"`
	} `mapstructure:"aliyun"`
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
