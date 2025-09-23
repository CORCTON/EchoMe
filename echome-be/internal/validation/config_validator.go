package validation

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/justin/echome-be/config"
)

// ConfigValidator 配置验证器
type ConfigValidator struct{}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateConfig 验证完整配置
func (v *ConfigValidator) ValidateConfig(cfg *config.Config) error {
	if err := v.validateServerConfig(cfg); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}

	if err := v.validateAIConfig(cfg); err != nil {
		return fmt.Errorf("AI config validation failed: %w", err)
	}

	if err := v.validateAliyunConfig(cfg); err != nil {
		return fmt.Errorf("Aliyun config validation failed: %w", err)
	}

	return nil
}

// validateServerConfig 验证服务器配置
func (v *ConfigValidator) validateServerConfig(cfg *config.Config) error {
	if cfg.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	// 验证端口格式
	if port, err := strconv.Atoi(cfg.Server.Port); err != nil {
		return fmt.Errorf("invalid server port format: %s", cfg.Server.Port)
	} else if port < 1 || port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got: %d", port)
	}

	return nil
}

// validateAIConfig 验证AI配置
func (v *ConfigValidator) validateAIConfig(cfg *config.Config) error {
	if cfg.AI.ServiceType == "" {
		return fmt.Errorf("AI service type is required")
	}

	supportedTypes := []string{"alibailian"}
	if !v.contains(supportedTypes, cfg.AI.ServiceType) {
		return fmt.Errorf("unsupported AI service type: %s, supported types: %s",
			cfg.AI.ServiceType, strings.Join(supportedTypes, ", "))
	}

	// 验证超时配置
	if cfg.AI.Timeout < 0 {
		return fmt.Errorf("AI timeout cannot be negative: %d", cfg.AI.Timeout)
	}

	if cfg.AI.MaxRetries < 0 {
		return fmt.Errorf("AI max retries cannot be negative: %d", cfg.AI.MaxRetries)
	}

	return nil
}

// validateAliyunConfig 验证阿里云配置
func (v *ConfigValidator) validateAliyunConfig(cfg *config.Config) error {
	if cfg.AI.ServiceType == "alibailian" {
		if cfg.APIKey == "" {
			return fmt.Errorf("Aliyun API key is required for alibailian service")
		}

		if cfg.Endpoint == "" {
			return fmt.Errorf("Aliyun endpoint is required")
		}

		// 验证endpoint格式
		if _, err := url.Parse(cfg.Endpoint); err != nil {
			return fmt.Errorf("invalid Aliyun endpoint format: %s", cfg.Endpoint)
		}

		if cfg.Region == "" {
			return fmt.Errorf("Aliyun region is required")
		}

		// 验证ASR配置
		if err := v.validateASRConfig(&cfg.ASR); err != nil {
			return fmt.Errorf("ASR config validation failed: %w", err)
		}

		// 验证TTS配置
		if err := v.validateTTSConfig(&cfg.TTS); err != nil {
			return fmt.Errorf("TTS config validation failed: %w", err)
		}

		// 验证LLM配置
		if err := v.validateLLMConfig(&cfg.LLM); err != nil {
			return fmt.Errorf("LLM config validation failed: %w", err)
		}
	}

	return nil
}

// validateASRConfig 验证ASR配置
func (v *ConfigValidator) validateASRConfig(asr *config.ASRServiceConfig) error {
	if asr.SampleRate <= 0 {
		return fmt.Errorf("ASR sample rate must be positive: %d", asr.SampleRate)
	}

	supportedFormats := []string{"pcm", "wav", "mp3"}
	if asr.Format != "" && !v.contains(supportedFormats, asr.Format) {
		return fmt.Errorf("unsupported ASR format: %s, supported: %s",
			asr.Format, strings.Join(supportedFormats, ", "))
	}

	return nil
}

// validateTTSConfig 验证TTS配置
func (v *ConfigValidator) validateTTSConfig(tts *config.TTSServiceConfig) error {
	if tts.SampleRate <= 0 {
		return fmt.Errorf("TTS sample rate must be positive: %d", tts.SampleRate)
	}

	supportedFormats := []string{"pcm", "wav", "mp3"}
	if tts.ResponseFormat != "" && !v.contains(supportedFormats, tts.ResponseFormat) {
		return fmt.Errorf("unsupported TTS response format: %s, supported: %s",
			tts.ResponseFormat, strings.Join(supportedFormats, ", "))
	}

	return nil
}

// validateLLMConfig 验证LLM配置
func (v *ConfigValidator) validateLLMConfig(llm *config.LLMServiceConfig) error {
	if llm.Temperature < 0 || llm.Temperature > 2 {
		return fmt.Errorf("LLM temperature must be between 0 and 2: %f", llm.Temperature)
	}

	if llm.MaxTokens <= 0 {
		return fmt.Errorf("LLM max tokens must be positive: %d", llm.MaxTokens)
	}

	return nil
}

// contains 检查切片是否包含指定元素
func (v *ConfigValidator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
