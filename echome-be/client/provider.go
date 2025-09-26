package client

import (
	"errors"

	"github.com/google/wire"
	"github.com/justin/echome-be/client/aliyun"
	"github.com/justin/echome-be/config"
	"github.com/justin/echome-be/internal/domain"
)

// Provider sets for Wire
var (
	// AIServiceProviderSet contains all AI service providers
	AIServiceProviderSet = wire.NewSet(
		NewAIServiceFromConfig,
		wire.Bind(new(domain.AIService), new(*aliyun.AliClient)),
	)
)

func NewAIServiceFromConfig(cfg *config.Config) (*aliyun.AliClient, error) {
	// Validate configuration
	if cfg.AI.ServiceType == "" {
		return nil, errors.New("AI service type is not configured")
	}
	// 使用配置中的所有相关参数
	aiService, err := NewAIService(
		AIServiceType(cfg.AI.ServiceType),
		cfg.Aliyun.APIKey,
		cfg.Aliyun.Endpoint,
		cfg.AI.Timeout,
		cfg.AI.MaxRetries,
		cfg.Aliyun.LLM.Model,
		cfg.Aliyun.LLM.MaxTokens,
		cfg.Aliyun.LLM.Temperature,
	)
	if err != nil {
		return nil, err
	}
	aliClient, ok := aiService.(*aliyun.AliClient)
	if !ok {
		return nil, errors.New("failed to convert to aliyun.AliClient")
	}
	return aliClient, nil
}
