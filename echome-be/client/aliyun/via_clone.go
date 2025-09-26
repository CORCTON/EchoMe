package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/justin/echome-be/internal/domain"
)

// VoiceCloneRequest 阿里云声音复刻API请求结构
type VoiceCloneRequest struct {
	Model string `json:"model"`
	Input struct {
		Action      string `json:"action"`       // 固定为create_voice
		TargetModel string `json:"target_model"` // 声音复刻使用的模型
		Prefix      string `json:"prefix"`       // 音色自定义前缀
		URL         string `json:"url"`          // 音频文件URL
		VoiceID     string `json:"voice_id"`     // 查询时使用的音色ID
	} `json:"input"`
}

// VoiceCloneAPIResponse 阿里云声音复刻API响应结构
type VoiceCloneAPIResponse struct {
	Output struct {
		VoiceID string `json:"voice_id"`
	} `json:"output"`
	Usage struct {
		Count int `json:"count"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

const (
	apiURL = "https://dashscope.aliyuncs.com/api/v1/services/audio/tts/customization"
)

// 默认克隆音色配置
func DefaultVoiceCloneTTSConfig() domain.TTSConfig {
	return domain.TTSConfig{
		Model:  "cosyvoice-v2",
		Voice:  "longxiaochun_v2",
		Format: "pcm",
	}
}

// 创建声音复刻专用的TTS配置
func CreateVoiceCloneTTSConfig(voiceID string, targetModel string) domain.TTSConfig {
	config := DefaultVoiceCloneTTSConfig()
	config.Voice = voiceID
	return config
}

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
func (client *AliClient) VoiceClone(ctx context.Context, config *domain.VoiceCloneConfig) (*string, error) {
	// 参数验证
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 验证音频URL
	if config.AudioURL == "" {
		return nil, fmt.Errorf("根据阿里云文档要求，必须提供公网可访问的音频URL")
	}

	// 固定音色前缀
	prefix := "echome"

	// 构建请求体
	requestBody := VoiceCloneRequest{
		Model: "voice-enrollment",
	}
	requestBody.Input.Action = "create_voice"
	requestBody.Input.TargetModel = config.TargetModel
	requestBody.Input.Prefix = prefix
	requestBody.Input.URL = config.AudioURL

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

// 使用角色来做 TTS
func (client *AliClient) TextToSpeechWithClone(ctx context.Context, text string, writer domain.WebSocketConn, c domain.Character) error {
	// 检查音色状态
	status, err := client.GetVoiceStatus(ctx, c.VoiceConfig.Voice)
	if err != nil {
		return fmt.Errorf("检查音色状态失败: %w", err)
	}

	if !status {
		return fmt.Errorf("音色审核中,请稍后重试")
	}

	// 创建TTS配置
	config := DefaultVoiceCloneTTSConfig()
	// 使用voice_id作为音色标识符
	config.Voice = c.VoiceConfig.Voice
	return client.HandleTTS(ctx, writer, text,config)
}
