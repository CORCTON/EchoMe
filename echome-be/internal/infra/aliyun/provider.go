package aliyun

import (
	"github.com/justin/echome-be/config"
)

// ProvideAliClient 创建阿里云百炼API客户端的提供者函数
func ProvideAliClient(cfg *config.Config) *AliClient {
	return NewAliClient(
		cfg.Aliyun.APIKey,
		cfg.Aliyun.Endpoint,
		cfg.AI.Timeout,
		cfg.AI.MaxRetries,
		cfg.Aliyun.LLM.Model,
		cfg.Aliyun.LLM.MaxTokens,
		cfg.Aliyun.LLM.Temperature,
		cfg.Tavily.APIKey,
	)
}
