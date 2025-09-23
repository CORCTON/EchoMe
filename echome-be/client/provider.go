package client

import (
	"errors"
	"fmt"

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

	// 根据配置选择AI服务类型
	serviceType := AIServiceType(cfg.AI.ServiceType)

	// 根据不同的服务类型创建对应的AI服务实例
	switch serviceType {
	case ServiceTypeALBL:
		// 使用统一的 Aliyun.APIKey 字段
		if cfg.Aliyun.APIKey == "" {
			return nil, errors.New("Aliyun API key is required")
		}

		// Endpoint 优先使用配置的 Aliyun.endpoint，否则回退到 ALBL.endpoint 再回退默认
		endpoint := cfg.Aliyun.Endpoint
		if endpoint == "" {
			endpoint = cfg.ALBL.Endpoint
		}
		if endpoint == "" {
			endpoint = "https://dashscope.aliyuncs.com"
		}

		client := aliyun.NewAliClient(cfg.Aliyun.APIKey, endpoint)
		if client == nil {
			return nil, errors.New("failed to create Aliyun client")
		}

		return client, nil
	default:
		return nil, fmt.Errorf("unsupported AI service type: %s", cfg.AI.ServiceType)
	}
}
