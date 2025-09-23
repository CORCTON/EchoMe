package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/justin/echome-be/internal/domain"
)

// AliClient encapsulates the client for Aliyun API
type AliClient struct {
	apiKey     string
	endPoint   string
	httpClient *http.Client
}

// RecognizeAudio implements domain.AIService.
func (client *AliClient) RecognizeAudio(ctx context.Context, audio []byte, config domain.ASRConfig) (string, error) {
	panic("unimplemented")
}

// SynthesizeSpeech implements domain.AIService.
func (client *AliClient) SynthesizeSpeech(ctx context.Context, text string, config domain.TTSConfig) ([]byte, error) {
	panic("unimplemented")
}

// 确保AliClient实现domain.AIService接口
var _ domain.AIService = (*AliClient)(nil)

func NewAliClient(apiKey string, endpoint string) *AliClient {
	return &AliClient{
		apiKey:   apiKey,
		endPoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// BailianRequest 阿里云百炼API请求结构
type BailianRequest struct {
	Model      string         `json:"model"`
	Input      BailianInput   `json:"input"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// BailianInput 输入结构
type BailianInput struct {
	Messages []BailianMessage `json:"messages"`
}

// BailianMessage 消息结构
type BailianMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// BailianResponse 阿里云百炼API响应结构
type BailianResponse struct {
	Output struct {
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
	} `json:"output"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
		InputTokens  int `json:"input_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// BailianErrorResponse 错误响应结构
type BailianErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func (client *AliClient) GenerateResponse(ctx context.Context, userInput string, characterContext string) (string, error) {
	// 输入验证
	if strings.TrimSpace(userInput) == "" {
		return "", fmt.Errorf("user input cannot be empty")
	}

	// 输入长度限制
	if len(userInput) > 4000 {
		return "", fmt.Errorf("user input too long (max 4000 characters)")
	}

	// 添加超时控制
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// 构建消息列表，支持角色上下文
	messages := []BailianMessage{}

	// 如果有角色上下文，添加系统消息
	if characterContext != "" {
		messages = append(messages, BailianMessage{
			Role:    "system",
			Content: fmt.Sprintf("你是一个AI助手，请根据以下角色设定进行对话：%s", characterContext),
		})
	}

	// 添加用户输入
	messages = append(messages, BailianMessage{
		Role:    "user",
		Content: userInput,
	})

	// 构建请求
	request := BailianRequest{
		Model: "qwen-turbo", // 使用通义千问模型
		Input: BailianInput{
			Messages: messages,
		},
		Parameters: map[string]interface{}{
			"result_format": "text",
			"max_tokens":    1500,
			"temperature":   0.7,
		},
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/api/v1/services/aigc/text-generation/generation", client.endPoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 发送请求（带重试机制）
	resp, err := client.doRequestWithRetry(req, 3)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		var errorResp BailianErrorResponse
		if err := json.Unmarshal(responseBody, &errorResp); err != nil {
			return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
		}
		return "", fmt.Errorf("API error [%s]: %s (request_id: %s)", errorResp.Code, errorResp.Message, errorResp.RequestID)
	}

	// 解析成功响应
	var response BailianResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查响应内容
	if response.Output.Text == "" {
		return "", fmt.Errorf("empty response from AI service")
	}

	return response.Output.Text, nil
}

// doRequestWithRetry 执行HTTP请求并支持重试
func (client *AliClient) doRequestWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		// 克隆请求以支持重试（因为body可能被消费）
		reqClone := req.Clone(req.Context())
		if req.Body != nil {
			// 重新设置body
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("failed to get request body: %w", err)
				}
				reqClone.Body = body
			}
		}

		resp, err := client.httpClient.Do(reqClone)
		if err == nil {
			// 检查是否需要重试（5xx错误或429）
			if resp.StatusCode < 500 && resp.StatusCode != 429 {
				return resp, nil
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		// 如果不是最后一次重试，等待一段时间
		if i < maxRetries {
			backoff := time.Duration(i+1) * time.Second
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}
