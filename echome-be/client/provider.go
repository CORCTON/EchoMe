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
	aiService, err := NewAIService(AIServiceType(cfg.AI.ServiceType), cfg.APIKey,cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	aliClient, ok := aiService.(*aliyun.AliClient)
	if !ok {
		return nil, errors.New("failed to convert to aliyun.AliClient")
	}
	return aliClient, nil
}
