package domain

import (
	"github.com/google/uuid"
)

// WebRTCService 定义WebRTC服务接口
 type WebRTCService interface {
	// CreatePeerConnection 创建新的对等连接
	CreatePeerConnection(sessionID uuid.UUID, userID string) (*PeerConnection, error)
	
	// HandleSignal 处理WebRTC信令消息
	HandleSignal(connectionID string, signal SignalMessage) error
	
	// ClosePeerConnection 关闭对等连接
	ClosePeerConnection(connectionID string) error
	
	// GetConnectionsBySession 获取会话的所有连接
	GetConnectionsBySession(sessionID uuid.UUID) ([]*PeerConnection, error)
}

// PeerConnection 表示WebRTC对等连接
 type PeerConnection struct {
	ID         string    `json:"id"`
	SessionID  uuid.UUID `json:"sessionId"`
	UserID     string    `json:"userId"`
	Status     string    `json:"status"` // "connecting", "connected", "disconnected"
	SocketConn WebSocketConn
}

// SignalMessage 表示WebRTC信令消息
 type SignalMessage struct {
	Type    string                 `json:"type"` // "offer", "answer", "ice-candidate"
	Target  string                 `json:"target,omitempty"`
	Payload map[string]interface{} `json:"payload"`
}