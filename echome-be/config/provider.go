package config

import (
	"github.com/google/wire"
)

var ConfigProviderSet = wire.NewSet(
	Load,
	GetDatabaseConfig,
	GetTavilyConfig,
)

func GetTavilyConfig(cfg *Config) *TavilyConfig {
	return &cfg.Tavily
}
