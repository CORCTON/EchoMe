package session

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
)

// sessionService implements domain.SessionService
type sessionService struct {
	sessionRepo      domain.SessionRepository
	messageRepo      domain.MessageRepository
	aiService        domain.AIService
	characterService domain.CharacterService
}

// NewSessionService creates a new session service
func NewSessionService(sessionRepo domain.SessionRepository, messageRepo domain.MessageRepository, aiService domain.AIService, characterService domain.CharacterService) domain.SessionService {
	return &sessionService{
		sessionRepo:      sessionRepo,
		messageRepo:      messageRepo,
		aiService:        aiService,
		characterService: characterService,
	}
}

// CreateSession creates a new chat session
func (s *sessionService) CreateSession(userID string, characterID uuid.UUID) (*domain.Session, error) {
	session := &domain.Session{
		ID:           uuid.New(),
		UserID:       userID,
		CharacterID:  characterID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	if err := s.sessionRepo.Save(session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSessionByID retrieves a session by ID
func (s *sessionService) GetSessionByID(id uuid.UUID) (*domain.Session, error) {
	return s.sessionRepo.GetByID(id)
}

// GetUserSessions retrieves all sessions for a user
func (s *sessionService) GetUserSessions(userID string) ([]*domain.Session, error) {
	return s.sessionRepo.GetByUserID(userID)
}

// SendMessage sends a message in a session
func (s *sessionService) SendMessage(sessionID uuid.UUID, content string, sender string) (*domain.Message, error) {
	message := &domain.Message{
		ID:        uuid.New(),
		SessionID: sessionID,
		Content:   content,
		Sender:    sender,
		Timestamp: time.Now(),
	}

	if err := s.messageRepo.Save(message); err != nil {
		return nil, err
	}

	// Update session last activity
	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return message, nil // Return message even if session update fails
	}

	session.LastActivity = time.Now()
	if err := s.sessionRepo.Save(session); err != nil {
		return message, err
	}

	// 生成AI回复（如果消息来自用户）
	if sender != "ai" {
		// 获取角色信息
		character, err := s.characterService.GetCharacterByID(session.CharacterID)
		if err != nil {
			return message, nil // 如果无法获取角色信息，只返回用户消息
		}

		// 构建角色上下文
		characterContext := "你是" + character.Name + "." + character.Description + "." + character.Persona

		// 生成AI回复
		aiResponse, err := s.aiService.GenerateResponse(context.Background(), content, characterContext)
		if err != nil {
			return message, nil // 如果生成回复失败，只返回用户消息
		}

		// 保存AI回复消息
		aiMessage := &domain.Message{
			ID:        uuid.New(),
			SessionID: sessionID,
			Content:   aiResponse,
			Sender:    "ai",
			Timestamp: time.Now(),
		}

		if err := s.messageRepo.Save(aiMessage); err != nil {
			return message, err
		}
	}

	return message, nil
}

// GetSessionMessages retrieves all messages in a session
func (s *sessionService) GetSessionMessages(sessionID uuid.UUID) ([]*domain.Message, error) {
	return s.messageRepo.GetBySessionID(sessionID)
}
