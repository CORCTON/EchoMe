package config

import (
	"os"

	"github.com/justin/echome-be/config/common"
)

// Config holds all application configuration
type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	WebRTC struct {
		STUNServer string `mapstructure:"stun_server"`
	} `mapstructure:"webrtc"`
	AI struct {
		ServiceType string `mapstructure:"service_type"`
		Timeout     int    `mapstructure:"timeout"`
		MaxRetries  int    `mapstructure:"max_retries"`
	} `mapstructure:"ai"`
	Aliyun `mapstructure:"aliyun"`
}

func Load(path string) *Config {
	_, c := common.MustLoad[Config](path)

	// 统一使用 ALIYUN_API_KEY 作为环境变量来源
	if apiKey := os.Getenv("ALIYUN_API_KEY"); apiKey != "" {
		c.APIKey = apiKey
	}

	return c
}