package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Message represents a chat message
type Message struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	Content   string
	Sender    string // 'user' or 'character'
	Timestamp time.Time
}

// MessageRepository defines the interface for message persistence
type MessageRepository interface {
	GetBySessionID(sessionID uuid.UUID) ([]*Message, error)
	Save(message *Message) error
}

// ConversationService provides voice conversation functionality
type ConversationService interface {
	// StartVoiceConversation 启动语音对话会话
	StartVoiceConversation(ctx context.Context, req *VoiceConversationRequest) error

	// ProcessTextMessage 处理文本消息并返回AI响应
	ProcessTextMessage(ctx context.Context, req *TextMessageRequest) (*TextMessageResponse, error)

	// GetCharacterVoiceConfig 获取角色的语音配置
	GetCharacterVoiceConfig(characterID uuid.UUID) (*VoiceConfig, error)
}

// VoiceConversationRequest represents a voice conversation request (simplified single-user mode)
type VoiceConversationRequest struct {
	WebSocketConn *websocket.Conn `json:"-"`
	CharacterID   uuid.UUID       `json:"character_id"`
	Language      string          `json:"language,omitempty"`
}

// ContextMessage 对话上下文中的单条消息

type ContextMessage struct {
	Role    string `json:"role"`    // "system", "user", or "assistant"
	Content string `json:"content"` // Message content
}

// TextMessageRequest 带上下文的文本消息请求

type TextMessageRequest struct {
	UserInput string           `json:"user_input"`
	UserID    string           `json:"user_id"`
	CharacterID string        `json:"character_id,omitempty"`
	Messages  []ContextMessage `json:"messages,omitempty"` // Conversation context
}

// TextMessageResponse 文本消息响应
type TextMessageResponse struct {
	Response  string    `json:"response"`
	MessageID uuid.UUID `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
}

// VoiceConfig 角色的语音配置
type VoiceConfig struct {
	Character *Character `json:"character"`
	ASRConfig ASRConfig  `json:"asr_config"`
	TTSConfig TTSConfig  `json:"tts_config"`
	Language  string     `json:"language"`
}