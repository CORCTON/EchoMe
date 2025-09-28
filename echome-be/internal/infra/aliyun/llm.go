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

	"github.com/justin/echome-be/internal/domain/ai"
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

// 定义搜索工具描述
func getSearchTool() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "perform_search",
			"description": "用于获取最新信息，回答需要联网获取的问题",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "搜索查询词",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// GenerateResponse LLM响应
func (client *AliClient) GenerateResponse(ctx context.Context, msg ai.DashScopeChatRequest, onChunk func(string) error) error {
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
	request := ai.DashScopeChatRequest{
		Messages: messages,
		Stream:   true,
	}

	if request.Model == "" {
		request.Model = model
	}

	// 如果启用了搜索功能，添加搜索工具
	if msg.EnableSearch {
		// 不再直接在请求中设置EnableSearch，而是通过工具调用让LLM决定
		request.Tools = append(request.Tools, getSearchTool())
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
			var chunk ai.DashScopeStreamChunk
			if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
				zap.L().Warn("Failed to unmarshal stream chunk", zap.Error(err), zap.String("json_data", jsonData))
				continue
			}

			// 处理内容块
			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]
				content := choice.Delta.Content
				toolCalls := choice.Delta.ToolCalls

				// 检查是否有工具调用请求
				if len(toolCalls) > 0 {
					zap.L().Info("Received tool call request", zap.Int("count", len(toolCalls)))

					// 处理工具调用
					for _, toolCall := range toolCalls {
						if toolCall.Name == "perform_search" && client.tavilyAPIKey != "" {
							// 获取搜索查询词
							query, ok := toolCall.Parameters["query"].(string)
							if !ok {
								zap.L().Error("Invalid search query parameter")
								continue
							}

							// 执行搜索
							zap.L().Info("Performing search", zap.String("query", query))
							searchContext, err := client.PerformSearchWithAPIKey(query)
							if err != nil {
								zap.L().Error("Search failed", zap.Error(err))
								if err := onChunk("搜索失败，请稍后再试。"); err != nil {
									return fmt.Errorf("callback error: %w", err)
								}
								continue
							}

							// 将搜索结果作为系统消息添加到会话
							messages = append(messages, map[string]any{
								"role":    "system",
								"content": "搜索结果：" + searchContext,
							})

							// 构建包含搜索结果的新请求
							toolResponseReq := ai.DashScopeChatRequest{
								Model:    model,
								Messages: messages,
								Stream:   true,
							}

							// 重新调用LLM生成响应
							if err := client.continueWithToolResponse(ctx, toolResponseReq, onChunk); err != nil {
								return err
							}

							// 处理完成后退出循环
							return nil
						}
					}
				}

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

// PerformSearchWithAPIKey 使用Tavily API执行搜索
func (client *AliClient) PerformSearchWithAPIKey(query string) (string, error) {
	// 定义Tavily搜索请求和响应结构
	tavilyAPIURL := "https://api.tavily.com/search"

	reqBody := map[string]any{
		"query":        query,
		"api_key":      client.tavilyAPIKey,
		"search_depth": "basic",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", tavilyAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search request failed with status: %d", resp.StatusCode)
	}

	var searchResult map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return "", fmt.Errorf("failed to decode search result: %w", err)
	}

	// 提取搜索结果文本
	var resultText strings.Builder
	if results, ok := searchResult["results"].([]interface{}); ok {
		for i, result := range results {
			if i >= 3 { // 限制返回结果数量
				break
			}
			if resultMap, ok := result.(map[string]interface{}); ok {
				if title, ok := resultMap["title"].(string); ok {
					resultText.WriteString("标题: " + title + "\n")
				}
				if snippet, ok := resultMap["snippet"].(string); ok {
					resultText.WriteString("摘要: " + snippet + "\n\n")
				}
			}
		}
	}

	return resultText.String(), nil
}

// continueWithToolResponse 处理带有工具调用结果的后续请求
func (client *AliClient) continueWithToolResponse(ctx context.Context, req ai.DashScopeChatRequest, onChunk func(string) error) error {
	// 构建请求体
	requestBody, err := json.Marshal(req)
	if err != nil {
		zap.L().Error("Failed to marshal request body", zap.Error(err))
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", client.endPoint, bytes.NewBuffer(requestBody))
	if err != nil {
		zap.L().Error("Failed to create HTTP request", zap.Error(err))
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+client.apiKey)

	// 执行请求（带重试）
	resp, err := client.doRequestWithRetry(httpReq, client.maxRetries)
	if err != nil {
		zap.L().Error("Request failed after retries", zap.Error(err))
		return fmt.Errorf("request failed after retries: %w", err)
	}
	defer resp.Body.Close()

	// 处理流式响应
	return client.handleStreamingResponse(ctx, resp, onChunk)
}

// handleStreamingResponse 处理流式响应
func (client *AliClient) handleStreamingResponse(ctx context.Context, resp *http.Response, onChunk func(string) error) error {
	reader := bufio.NewReader(resp.Body)

	for {
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
			var chunk ai.DashScopeStreamChunk
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

	zap.L().Info("Streaming response processing completed for tool response")

	return nil
}
