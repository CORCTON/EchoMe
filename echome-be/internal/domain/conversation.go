package domain

import (
	"context"

	"github.com/google/uuid"
)

// ConversationService 对话服务
type ConversationService interface {
	// StartVoiceConversation 启动语音对话会话
	StartVoiceConversation(ctx context.Context, req *VoiceConversationRequest) error
}

// VoiceConversationRequest 语音对话请求
type VoiceConversationRequest struct {
	SafeConn    WebSocketConn `json:"-"`
	CharacterID uuid.UUID     `json:"character_id"`
}

// ContextMessage 对话上下文中的单条消息
type ContextMessage struct {
	Role    string `json:"role"`    // "system", "user", or "assistant"
	Content string `json:"content"` // 历史记录
}

// VoiceConfig 角色的语音配置
type VoiceConfig struct {
	Character *Character `json:"character"`
	ASRConfig ASRConfig  `json:"asr_config"`
	TTSConfig TTSConfig  `json:"tts_config"`
	Language  string     `json:"language"`
}
