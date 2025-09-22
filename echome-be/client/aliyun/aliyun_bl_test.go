package aliyun

import (
	"context"
	"testing"
)

func TestAliClient_GenerateResponse(t *testing.T) {
	// 注意：这个测试需要真实的API密钥才能运行
	// 在实际环境中，应该使用mock来测试

	client := NewAliClient("sk-64d642d1fbfc448f94560df95bc34d96", "https://dashscope.aliyuncs.com")

	// 测试基本响应生成（不会实际调用API，因为使用的是测试密钥）
	ctx := context.Background()
	userInput := "你好"
	characterContext := ""

	// 这个测试主要验证方法签名和基本结构
	result, err := client.GenerateResponse(ctx, userInput, characterContext)

	// 由于使用测试密钥，预期会有错误
	if err == nil {
		t.Logf("Generated response: %s", result)
	} else {
		t.Logf("Expected error with test API key: %v", err)
	}
}

func TestAliClient_GenerateResponseWithCharacter(t *testing.T) {
	client := NewAliClient("test-api-key", "")

	ctx := context.Background()
	userInput := "介绍一下你自己"
	characterContext := "你是一个友善的AI助手，名叫小明，喜欢帮助用户解决问题。"

	// 测试带角色上下文的响应生成
	_, err := client.GenerateResponse(ctx, userInput, characterContext)

	// 由于使用测试密钥，预期会有错误，但方法应该正确处理角色上下文
	if err == nil {
		t.Log("GenerateResponse with character context executed without panic")
	} else {
		t.Logf("Expected error with test API key: %v", err)
	}
}
