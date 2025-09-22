package webrtc

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
)

// WebRTCService 实现domain.WebRTCService接口
type webRTCService struct {
	connections        map[string]*domain.PeerConnection
	sessionConnections map[uuid.UUID]map[string]*domain.PeerConnection
	mutex              sync.RWMutex
}

// NewWebRTCService 创建新的WebRTC服务实例
func NewWebRTCService() domain.WebRTCService {
	return &webRTCService{
		connections:        make(map[string]*domain.PeerConnection),
		sessionConnections: make(map[uuid.UUID]map[string]*domain.PeerConnection),
	}
}

// CreatePeerConnection 创建新的对等连接
func (s *webRTCService) CreatePeerConnection(sessionID uuid.UUID, userID string) (*domain.PeerConnection, error) {
	connectionID := uuid.New().String()

	conn := &domain.PeerConnection{
		ID:        connectionID,
		SessionID: sessionID,
		UserID:    userID,
		Status:    "connecting",
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.connections[connectionID] = conn

	// 确保会话连接映射存在
	if _, exists := s.sessionConnections[sessionID]; !exists {
		s.sessionConnections[sessionID] = make(map[string]*domain.PeerConnection)
	}
	s.sessionConnections[sessionID][connectionID] = conn

	return conn, nil
}

// HandleSignal 处理WebRTC信令消息
func (s *webRTCService) HandleSignal(connectionID string, signal domain.SignalMessage) error {
	s.mutex.RLock()
	_, exists := s.connections[connectionID]
	s.mutex.RUnlock()

	if !exists {
		return errors.New("connection not found")
	}

	// 这里可以实现信令消息的处理逻辑
	// 例如转发给目标连接
	if signal.Target != "" && signal.Target != connectionID {
		s.mutex.RLock()
		targetConn, targetExists := s.connections[signal.Target]
		s.mutex.RUnlock()

		if targetExists && targetConn.SocketConn != nil {
			// 转发信令消息给目标连接
			response := map[string]interface{}{
				"type":    signal.Type,
				"from":    connectionID,
				"payload": signal.Payload,
			}

			err := targetConn.SocketConn.WriteJSON(response)
			if err != nil {
				return fmt.Errorf("failed to forward signal: %w", err)
			}
		}
	}

	return nil
}

// ClosePeerConnection 关闭对等连接
func (s *webRTCService) ClosePeerConnection(connectionID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	connection, exists := s.connections[connectionID]
	if !exists {
		return errors.New("connection not found")
	}

	// 关闭WebSocket连接
	if connection.SocketConn != nil {
		connection.SocketConn.Close()
	}

	// 从映射中删除连接
	sessionID := connection.SessionID
	delete(s.connections, connectionID)

	// 从会话连接映射中删除
	if sessionConns, exists := s.sessionConnections[sessionID]; exists {
		delete(sessionConns, connectionID)

		// 如果会话没有连接了，删除会话映射
		if len(sessionConns) == 0 {
			delete(s.sessionConnections, sessionID)
		}
	}

	return nil
}

// GetConnectionsBySession 获取会话的所有连接
func (s *webRTCService) GetConnectionsBySession(sessionID uuid.UUID) ([]*domain.PeerConnection, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sessionConns, exists := s.sessionConnections[sessionID]
	if !exists {
		return []*domain.PeerConnection{}, nil
	}

	connections := make([]*domain.PeerConnection, 0, len(sessionConns))
	for _, conn := range sessionConns {
		connections = append(connections, conn)
	}

	return connections, nil
}
