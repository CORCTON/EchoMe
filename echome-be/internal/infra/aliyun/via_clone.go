package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetVoiceStatus 根据音色ID查询音色状态
func (client *AliClient) GetVoiceStatus(ctx context.Context, voiceID string) (bool, error) {
	// 参数验证
	if voiceID == "" {
		return false, fmt.Errorf("音色ID不能为空")
	}

	// 构建请求体
	requestBody := VoiceCloneRequest{
		Model: "voice-enrollment",
	}
	requestBody.Input.Action = "query_voice"
	requestBody.Input.VoiceID = voiceID

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.apiKey)

	// 发送请求
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 解析响应
	var listResponse map[string]interface{}
	if err := json.Unmarshal(responseBody, &listResponse); err != nil {
		return false, fmt.Errorf("解析响应失败: %w，原始响应: %s", err, string(responseBody))
	}

	// 提取voices数组
	output, ok := listResponse["output"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	voices, ok := output["status"]
	if !ok {
		return false, nil
	}
	if voices == "OK" {
		return true, nil
	}
	return false, nil
}

// 克隆声音接口
func (client *AliClient) VoiceClone(ctx context.Context, url string) (*string, error) {
	// 参数验证
	if url == "" {
		return nil, fmt.Errorf("音频URL不能为空")
	}

	// 固定音色前缀
	prefix := "echome"

	// 构建请求体
	requestBody := VoiceCloneRequest{
		Model: "voice-enrollment",
	}
	requestBody.Input.Action = "create_voice"
	// 固定模型
	requestBody.Input.TargetModel = "cosyvoice-v2"
	requestBody.Input.Prefix = prefix
	requestBody.Input.URL = url

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.apiKey)

	// 发送请求
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 解析响应
	var apiResponse VoiceCloneAPIResponse
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w，原始响应: %s", err, string(responseBody))
	}

	// 检查是否返回了voice_id
	if apiResponse.Output.VoiceID == "" {
		return nil, fmt.Errorf("未返回voice_id")
	}

	return &apiResponse.Output.VoiceID, nil
}
