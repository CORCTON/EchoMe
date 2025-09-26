package aliyun

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

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

func (client *AliClient) GenerateResponse(ctx context.Context, userInput string, characterContext string, conversationHistory []map[string]string) (string, error) {
	// 输入验证
	if strings.TrimSpace(userInput) == "" {
		return "", fmt.Errorf("user input cannot be empty")
	}

	// 输入长度限制
	if len(userInput) > 4000 {
		return "", fmt.Errorf("user input too long (max 4000 characters)")
	}

	// 添加超时控制
	timeout := 30 * time.Second
	if client.timeout > 0 {
		timeout = time.Duration(client.timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 构建消息列表，支持角色上下文和对话历史
	messages := []BailianMessage{}

	// 如果有角色上下文，添加系统消息
	if characterContext != "" {
		messages = append(messages, BailianMessage{
			Role:    "system",
			Content: fmt.Sprintf("你是一个AI助手，请根据以下角色设定进行对话：%s", characterContext),
		})
	}

	// 添加对话历史消息
	for _, msg := range conversationHistory {
		role, roleOk := msg["role"]
		content, contentOk := msg["content"]
		// 确保角色和内容都存在，且角色是合法的
		if roleOk && contentOk && (role == "user" || role == "assistant") {
			messages = append(messages, BailianMessage{
				Role:    role,
				Content: content,
			})
		}
	}

	// 添加最新的用户输入
	messages = append(messages, BailianMessage{
		Role:    "user",
		Content: userInput,
	})

	// 使用配置的LLM参数
	model := "qwen-turbo"       // 默认值
	maxTokens := 1500           // 默认值
	temperature := float32(0.7)

	if client.llmModel != "" {
		model = client.llmModel
	}
	if client.maxTokens > 0 {
		maxTokens = client.maxTokens
	}
	if client.temperature > 0 {
		temperature = client.temperature
	}

	// 构建请求
	request := BailianRequest{
		Model: model,
		Input: BailianInput{
			Messages: messages,
		},
		Parameters: map[string]interface{}{
			"result_format": "text",
			"max_tokens":    maxTokens,
			"temperature":   float64(temperature), // 转换为float64
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
	retries := 3
	if client.maxRetries > 0 {
		retries = client.maxRetries
	}
	resp, err := client.doRequestWithRetry(req, retries)
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

// DashScopeChatRequest 阿里云DashScope请求结构
type DashScopeChatRequest struct {
	Model    string              `json:"model"`
	Messages []map[string]string `json:"messages"`
	Stream   bool                `json:"stream"`
}

// DashScopeStreamChunk DashScope流式响应块结构
type DashScopeStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content,omitempty"`
			Role    string `json:"role,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
}

// GenerateStreamResponse 生成AI流式响应（使用阿里云DashScope兼容模式）
func (client *AliClient) GenerateStreamResponse(ctx context.Context, userInput string, characterContext string, conversationHistory []map[string]string, onChunk func(string) error) error {
	// 输入验证
	if strings.TrimSpace(userInput) == "" {
		return fmt.Errorf("user input cannot be empty")
	}

	// 输入长度限制
	if len(userInput) > 4000 {
		return fmt.Errorf("user input too long (max 4000 characters)")
	}

	// 添加超时控制
	timeout := 30 * time.Second
	if client.timeout > 0 {
		timeout = time.Duration(client.timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 使用配置的LLM参数
	model := "qwen-plus"       // 默认使用qwen-plus模型
	if client.llmModel != "" {
		model = client.llmModel
	}

	// 构建消息列表，支持角色上下文和对话历史
	var messages []map[string]string

	// 如果有角色上下文，添加系统消息
	if characterContext != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": fmt.Sprintf("你是一个AI助手，请根据以下角色设定进行对话：%s", characterContext),
		})
	}

	// 添加对话历史消息
	for _, msg := range conversationHistory {
		role, roleOk := msg["role"]
		content, contentOk := msg["content"]
		// 确保角色和内容都存在，且角色是合法的
		if roleOk && contentOk && (role == "user" || role == "assistant") {
			messages = append(messages, map[string]string{
				"role":    role,
				"content": content,
			})
		}
	}

	// 添加最新的用户输入
	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userInput,
	})

	// 构建请求
	request := DashScopeChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	url := "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer " + client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	// 发送请求
	resp, err := client.doRequestWithRetry(req, 3)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		responseBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("API request failed with status %d and could not read response: %w", resp.StatusCode, readErr)
		}
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 处理流式响应
	reader := bufio.NewReader(resp.Body)
	zap.L().Info("Starting to process streaming response using DashScope compatible mode")

	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			zap.L().Info("Streaming context canceled")
			return ctx.Err()
		default:
		}

		// 读取一行数据
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// 流结束
				zap.L().Info("Reached end of streaming response")
				break
			}
			zap.L().Error("Error reading stream line", zap.Error(err))
			return fmt.Errorf("failed to read stream: %w", err)
		}

		// 记录原始数据行（调试用）
		zap.L().Debug("Received stream line", zap.String("line", line))

		// 跳过空行
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否是数据行（固定格式：data: 开头）
		const dataPrefix = "data: "
		if strings.HasPrefix(line, dataPrefix) {
			// 提取JSON数据（直接去掉固定前缀）
			jsonData := line[len(dataPrefix):]

			// 检查是否是结束标记
			if jsonData == "[DONE]" {
				break
			}

			// 解析JSON
			var chunk DashScopeStreamChunk
			if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
				zap.L().Warn("Failed to unmarshal stream chunk", zap.Error(err), zap.String("json_data", jsonData))
				continue
			}

			// 处理内容块
				if len(chunk.Choices) > 0 {
					choice := chunk.Choices[0]
					content := choice.Delta.Content

					// 如果有文本内容，通过回调函数返回
					if content != "" {
						if err := onChunk(content); err != nil {
							return fmt.Errorf("callback error: %w", err)
						}
					}
			}
		}
	}

	zap.L().Info("Streaming response processing completed using DashScope compatible mode")

	return nil
}
