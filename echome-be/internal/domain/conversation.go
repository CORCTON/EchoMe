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

// Session represents a chat session between a user and a character
type Session struct {
	ID           uuid.UUID
	UserID       string
	CharacterID  uuid.UUID
	CreatedAt    time.Time
	LastActivity time.Time
}

// SessionRepository defines the interface for session persistence
type SessionRepository interface {
	GetByID(id uuid.UUID) (*Session, error)
	GetByUserID(userID string) ([]*Session, error)
	Save(session *Session) error
}

// MessageRepository defines the interface for message persistence
type MessageRepository interface {
	GetBySessionID(sessionID uuid.UUID) ([]*Message, error)
	Save(message *Message) error
}

// SessionService provides business logic for session operations
type SessionService interface {
	CreateSession(userID string, characterID uuid.UUID) (*Session, error)
	GetSessionByID(id uuid.UUID) (*Session, error)
	GetUserSessions(userID string) ([]*Session, error)
	SendMessage(sessionID uuid.UUID, content string, sender string) (*Message, error)
	GetSessionMessages(sessionID uuid.UUID) ([]*Message, error)
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

// VoiceConversationRequest represents a voice conversation request
type VoiceConversationRequest struct {
	SessionID     uuid.UUID       `json:"session_id"`
	CharacterID   uuid.UUID       `json:"character_id"`
	WebSocketConn *websocket.Conn `json:"-"`
	UserID        string          `json:"user_id"`
	Language      string          `json:"language,omitempty"`
}

// TextMessageRequest represents a text message request
type TextMessageRequest struct {
	SessionID   uuid.UUID `json:"session_id"`
	CharacterID uuid.UUID `json:"character_id"`
	UserInput   string    `json:"user_input"`
	UserID      string    `json:"user_id"`
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

// AIRequest represents a request to AI service with context
type AIRequest struct {
	UserInput        string     `json:"user_input"`
	CharacterContext *Character `json:"character_context"`
	SessionHistory   []*Message `json:"session_history"`
	Language         string     `json:"language"`
}

// AIResponse represents AI service response
type AIResponse struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
}
