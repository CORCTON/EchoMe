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

	"github.com/justin/echome-be/internal/domain"
	"go.uber.org/zap"
)

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

// GenerateResponse LLM响应
func (client *AliClient) GenerateResponse(ctx context.Context, msg domain.DashScopeChatRequest, onChunk func(string) error) error {
	// 添加超时控制
	timeout := 30 * time.Second
	if client.timeout > 0 {
		timeout = time.Duration(client.timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 使用配置的LLM参数
	model := client.llmModel
	if model == "" {
		model = "qwen3-vl-plus"
	}

	// 验证并转换Messages
	var messages []map[string]any
	for i, cm := range msg.Messages {
		// 检查cm是否为nil
		if cm == nil {
			zap.L().Warn("Skipping nil message", zap.Int("index", i))
			continue
		}

		// 检查必填字段
		role, roleOk := cm["role"]
		content, contentOk := cm["content"]
		if !roleOk || !contentOk {
			zap.L().Warn("Skipping message with missing required fields", zap.Int("index", i))
			continue
		}

		// 确保content不为空
		contentStr, ok := content.(string)
		if ok && contentStr == "" {
			zap.L().Warn("Skipping message with empty content", zap.Int("index", i))
			continue
		}

		message := map[string]any{
			"role":    role,
			"content": content,
		}
		messages = append(messages, message)
	}

	// 构建请求
	request := domain.DashScopeChatRequest{
		Messages:     messages,
		Stream:       true,
	}

	if request.Model == "" {
		request.Model = model
	}
	if msg.EnableSearch {
		request.EnableSearch = true
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	zap.L().Warn("Request body", zap.String("body", string(requestBody)))
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
	req.Header.Set("Authorization", "Bearer "+client.apiKey)
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
			var chunk domain.DashScopeStreamChunk
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
