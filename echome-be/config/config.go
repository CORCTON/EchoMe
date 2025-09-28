package config

// Config holds all application configuration
type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	AI struct {
		ServiceType string `mapstructure:"service_type"`
		Timeout     int    `mapstructure:"timeout"`
		MaxRetries  int    `mapstructure:"max_retries"`
	} `mapstructure:"ai"`
	Aliyun   Aliyun         `mapstructure:"aliyun"`
	Tavily   TavilyConfig   `mapstructure:"tavily"`
	Database DatabaseConfig `mapstructure:"database"`
}

// TavilyConfig holds Tavily API configuration
type TavilyConfig struct {
	APIKey string `mapstructure:"api_key"`
}

func Load(path string) (*Config, error) {
	_, c := MustLoad[Config](path)
	return c, nil
}
