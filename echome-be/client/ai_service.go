package client

import (
	"errors"

	"github.com/justin/echome-be/client/aliyun"
	"github.com/justin/echome-be/internal/domain"
)

// AIServiceType 定义AI服务类型
type AIServiceType string

const (
	// ServiceTypeALBL 阿里云百炼服务
	ServiceTypeALBL AIServiceType = "alibailian"
)

// NewAIService 根据服务类型创建对应的AI服务实例
func NewAIService(serviceType AIServiceType, apiKey string, optionalParams ...string) (domain.AIService, error) {
	switch serviceType {
	case ServiceTypeALBL:
		endpoint := ""
		if len(optionalParams) > 1 {
			endpoint = optionalParams[1]
		}
		return aliyun.NewAliClient(apiKey, endpoint), nil
	default:
		return nil, errors.New("unknown AI service type")
	}
}
