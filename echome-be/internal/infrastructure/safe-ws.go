package infrastructure

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
)

var _ domain.WSWriter = (*SafeConn)(nil) // 确保 SafeConn 实现了接口

// SafeConn 保证 WebSocket 写操作串行化
type SafeConn struct {
	conn     *websocket.Conn
	writeCh  chan func() error
	closed   chan struct{}
	once     sync.Once
	closeErr error
}

// NewSafeConn 创建安全的连接包装
func NewSafeConn(conn *websocket.Conn) *SafeConn {
	sc := &SafeConn{
		conn:    conn,
		writeCh: make(chan func() error, 100), // 缓冲防止阻塞
		closed:  make(chan struct{}),
	}
	go sc.writeLoop()
	return sc
}

// writeLoop 串行消费写队列
func (sc *SafeConn) writeLoop() {
	defer close(sc.closed)
	for fn := range sc.writeCh {
		if err := fn(); err != nil {
			log.Printf("SafeConn write error: %v", err)
			sc.closeErr = err
			_ = sc.conn.Close()
			return
		}
	}
}

// WriteJSON 安全写 JSON 消息
func (sc *SafeConn) WriteJSON(v any) error {
	select {
	case sc.writeCh <- func() error { return sc.conn.WriteJSON(v) }:
		return nil
	case <-sc.closed:
		return errors.New("connection already closed")
	}
}

// WriteMessage 安全写文本或二进制消息
func (sc *SafeConn) WriteMessage(messageType int, data []byte) error {
	select {
	case sc.writeCh <- func() error { return sc.conn.WriteMessage(messageType, data) }:
		return nil
	case <-sc.closed:
		return errors.New("connection already closed")
	}
}

// WriteJSONCtx 支持超时控制的安全写 JSON
func (sc *SafeConn) WriteJSONCtx(ctx context.Context, v any) error {
	select {
	case sc.writeCh <- func() error { return sc.conn.WriteJSON(v) }:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-sc.closed:
		return errors.New("connection already closed")
	}
}

// Close 关闭连接
func (sc *SafeConn) Close() error {
	sc.once.Do(func() {
		close(sc.writeCh) // 停止写
	})
	<-sc.closed
	return sc.conn.Close()
}

// CloseWithError 关闭连接并记录错误原因
func (sc *SafeConn) CloseWithError(err error) error {
	sc.closeErr = err
	return sc.Close()
}

// Err 返回连接关闭时的错误原因
func (sc *SafeConn) Err() error {
	return sc.closeErr
}

// Underlying 返回底层原始 websocket.Conn（用于读）
func (sc *SafeConn) Underlying() *websocket.Conn {
	return sc.conn
}
