package infrastructure

import (
	"errors"
	"slices"
	"sync"

	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
)

type MemorySessionRepository struct {
	sessions  map[uuid.UUID]*domain.Session
	userIndex map[string][]uuid.UUID
	mu        sync.RWMutex
}

func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{
		sessions:  make(map[uuid.UUID]*domain.Session),
		userIndex: make(map[string][]uuid.UUID),
	}
}

// GetByID 根据ID获取会话
func (r *MemorySessionRepository) GetByID(id uuid.UUID) (*domain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[id]
	if !exists {
		return nil, errors.New("session not found")
	}

	return session, nil
}

// GetByUserID 根据用户ID获取会话列表
func (r *MemorySessionRepository) GetByUserID(userID string) ([]*domain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids, exists := r.userIndex[userID]
	if !exists {
		return []*domain.Session{}, nil
	}

	var sessions []*domain.Session
	for _, id := range ids {
		if session, exists := r.sessions[id]; exists {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// Save 新建会话
func (r *MemorySessionRepository) Save(session *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.ID] = session

	// Update user index
	if _, exists := r.userIndex[session.UserID]; !exists {
		r.userIndex[session.UserID] = []uuid.UUID{session.ID}
	} else {
		// Check if session ID already exists in the user's list
		exists := slices.Contains(r.userIndex[session.UserID], session.ID)
		if !exists {
			r.userIndex[session.UserID] = append(r.userIndex[session.UserID], session.ID)
		}
	}

	return nil
}

type MemoryMessageRepository struct {
	messages     map[uuid.UUID]*domain.Message
	sessionIndex map[uuid.UUID][]uuid.UUID
	mu           sync.RWMutex
}

func NewMemoryMessageRepository() *MemoryMessageRepository {
	return &MemoryMessageRepository{
		messages:     make(map[uuid.UUID]*domain.Message),
		sessionIndex: make(map[uuid.UUID][]uuid.UUID),
	}
}

// GetBySessionID 根据会话ID获取消息列表
func (r *MemoryMessageRepository) GetBySessionID(sessionID uuid.UUID) ([]*domain.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids, exists := r.sessionIndex[sessionID]
	if !exists {
		return []*domain.Message{}, nil
	}

	var messages []*domain.Message
	for _, id := range ids {
		if message, exists := r.messages[id]; exists {
			messages = append(messages, message)
		}
	}

	return messages, nil
}

// Save 新建消息
func (r *MemoryMessageRepository) Save(message *domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messages[message.ID] = message

	// Update session index
	if _, exists := r.sessionIndex[message.SessionID]; !exists {
		r.sessionIndex[message.SessionID] = []uuid.UUID{message.ID}
	} else {
		r.sessionIndex[message.SessionID] = append(r.sessionIndex[message.SessionID], message.ID)
	}

	return nil
}
