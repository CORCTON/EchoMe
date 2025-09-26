package domain

import (
	"time"
)

// WebSocketConn 定义统一的WebSocket连接接口，封装所有WebSocket操作
type WebSocketConn interface {
	// ReadJSON 从WebSocket读取JSON数据
	ReadJSON(v any) error

	// ReadMessage 从WebSocket读取消息
	ReadMessage() (messageType int, p []byte, err error)

	// WriteJSON 将JSON数据写入WebSocket
	WriteJSON(v any) error

	// WriteMessage 将消息写入WebSocket
	WriteMessage(msgType int, data []byte) error

	// Close 关闭WebSocket连接
	Close() error

	// SetReadDeadline 设置读取截止时间
	SetReadDeadline(t time.Time) error

	// SetWriteDeadline 设置写入截止时间
	SetWriteDeadline(t time.Time) error

	// SetPongHandler 设置Pong消息处理器
	SetPongHandler(h func(string) error)
}
