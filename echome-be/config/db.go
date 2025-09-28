package config

import "fmt"

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     uint16 `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"db_name"`
}

func GetDatabaseConfig(cfg *Config) *DatabaseConfig {
	if cfg == nil {
		panic("config is nil")
	}
	return &cfg.Database
}

const postgresTCPDSN = "host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai"

func (cfg *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		postgresTCPDSN,
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.Port,
	)
}
