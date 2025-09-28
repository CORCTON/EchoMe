package aliyun

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/justin/echome-be/internal/domain/ai"
)

// 确保AliClient实现domain.AIRepo接口
var _ ai.Repo = (*AliClient)(nil)

// AliClient 阿里云百炼API客户端
type AliClient struct {
	apiKey       string
	endPoint     string
	timeout      int
	maxRetries   int
	httpClient   *http.Client
	llmModel     string
	maxTokens    int
	temperature  float32
	tavilyAPIKey string
}

func NewAliClient(apiKey string, endpoint string, timeout int, maxRetries int, llmModel string, maxTokens int, temperature float32, tavilyAPIKey string) *AliClient {
	// 为超时配置设置默认值
	httpTimeout := 30 * time.Second
	if timeout > 0 {
		httpTimeout = time.Duration(timeout) * time.Second
	}

	return &AliClient{
		apiKey:       apiKey,
		endPoint:     endpoint,
		timeout:      timeout,
		maxRetries:   maxRetries,
		llmModel:     llmModel,
		maxTokens:    maxTokens,
		temperature:  temperature,
		tavilyAPIKey: tavilyAPIKey,
		httpClient: &http.Client{
			Timeout: httpTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // 临时禁用TLS验证
				},
			},
		},
	}
}
