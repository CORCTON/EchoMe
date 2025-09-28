package conversation

import (
	"fmt"
)

// ConversationError 会话错误
type ConversationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ConversationError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ErrorCodes 会话错误码
const (
	ErrCodeCharacterNotFound  = "CHARACTER_NOT_FOUND"
	ErrCodeSessionNotFound    = "SESSION_NOT_FOUND"
	ErrCodeASRFailed          = "ASR_FAILED"
	ErrCodeTTSFailed          = "TTS_FAILED"
	ErrCodeAIGenerationFailed = "AI_GENERATION_FAILED"
	ErrCodeWebSocketError     = "WEBSOCKET_ERROR"
	ErrCodeInvalidInput       = "INVALID_INPUT"
	ErrCodeConfigurationError = "CONFIGURATION_ERROR"
	ErrCodeMessageSaveFailed  = "MESSAGE_SAVE_FAILED"
)

// NewConversationError 创建会话错误
func NewConversationError(code, message, details string) *ConversationError {
	return &ConversationError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapError 包装错误
func WrapError(code, message string, err error) *ConversationError {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return NewConversationError(code, message, details)
}
