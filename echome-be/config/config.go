package config

import (
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
	Aliyun Aliyun `mapstructure:"aliyun"`
	Database DatabaseConfig `mapstructure:"database"`
}

func Load(path string) *Config {
	_, c := common.MustLoad[Config](path)
	return c
}