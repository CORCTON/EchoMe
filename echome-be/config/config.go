package config

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
	Aliyun   Aliyun         `mapstructure:"aliyun"`
	Database DatabaseConfig `mapstructure:"database"`
}

func Load(path string) (*Config, error) {
	_, c := MustLoad[Config](path)
	return c, nil
}
