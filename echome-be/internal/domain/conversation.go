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
	// StartVoiceConversation starts a voice conversation session
	StartVoiceConversation(ctx context.Context, req *VoiceConversationRequest) error

	// ProcessTextMessage processes a text message and returns AI response
	ProcessTextMessage(ctx context.Context, req *TextMessageRequest) (*TextMessageResponse, error)

	// GetCharacterVoiceConfig retrieves voice configuration for a character
	GetCharacterVoiceConfig(characterID uuid.UUID) (*VoiceConfig, error)
}

// VoiceConversationRequest represents a voice conversation request (simplified single-user mode)
type VoiceConversationRequest struct {
	WebSocketConn *websocket.Conn `json:"-"`
	CharacterID   uuid.UUID       `json:"character_id"`
	Language      string          `json:"language,omitempty"`
}

// TextMessageRequest represents a text message request (simplified)
type TextMessageRequest struct {
	UserInput string `json:"user_input"`
	UserID    string `json:"user_id"`
}

// TextMessageResponse represents a text message response
type TextMessageResponse struct {
	Response  string    `json:"response"`
	MessageID uuid.UUID `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
}

// VoiceConfig represents voice conversation configuration
type VoiceConfig struct {
	Character *Character `json:"character"`
	ASRConfig ASRConfig  `json:"asr_config"`
	TTSConfig TTSConfig  `json:"tts_config"`
	Language  string     `json:"language"`
}

// AIRequest represents a request to AI service with context (simplified)
type AIRequest struct {
	UserInput        string     `json:"user_input"`
	CharacterContext *Character `json:"character_context"`
	Language         string     `json:"language"`
}

// AIResponse represents AI service response
type AIResponse struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
}
