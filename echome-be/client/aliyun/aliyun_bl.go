package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/justin/echome-be/internal/domain"
)

// AliClient encapsulates the client for Aliyun API
type AliClient struct {
	apiKey   string
	endPoint string
}

// 确保AliClient实现domain.AIService接口
var _ domain.AIService = (*AliClient)(nil)

func NewAliClient(apiKey string, endpoint string) *AliClient {
	if endpoint == "" {
		endpoint = "https://dashscope.aliyuncs.com"
	}
	return &AliClient{apiKey: apiKey, endPoint: endpoint}
}

// BailianRequest 阿里云百炼API请求结构
type BailianRequest struct {
	Model      string                 `json:"model"`
	Input      BailianInput           `json:"input"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
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

	// 发送请求
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := httpClient.Do(req)
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
