package infra

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
)

// --- 心跳相关的常量 ---
const (
	// writeWait 是允许向对端写入消息的时间。
	writeWait = 10 * time.Second
	// pongWait 是等待对端响应 Pong 消息的时间。必须大于 pingPeriod。
	pongWait = 60 * time.Second
	// pingPeriod 是向对端发送 Ping 消息的周期。
	pingPeriod = (pongWait * 9) / 10
)

var _ domain.WebSocketConn = (*SafeConn)(nil) // 确保 SafeConn 实现了接口

// SafeConn 保证 WebSocket 写操作串行化，并内置心跳机制
type SafeConn struct {
	conn     *websocket.Conn
	writeCh  chan func() error
	closed   chan struct{}
	once     sync.Once
	closeErr error
}

// NewSafeConn 创建安全的连接包装，并自动启动心跳
func NewSafeConn(conn *websocket.Conn) *SafeConn {
	sc := &SafeConn{
		conn:    conn,
		writeCh: make(chan func() error, 100), // 缓冲防止阻塞
		closed:  make(chan struct{}),
	}

	// --- 启动心跳机制 ---
	// 1. 设置初始的读超时
	if err := sc.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("NewSafeConn: Failed to set initial read deadline: %v", err)
	}

	// 2. 设置 Pong 处理器来延长读超时
	sc.SetPongHandler(func(string) error {
		log.Println("Pong received, extending read deadline.")
		return sc.SetReadDeadline(time.Now().Add(pongWait))
	})

	// 3. 启动后台的写循环和 Ping 循环
	go sc.writeLoop()
	go sc.pingLoop()

	return sc
}

// writeLoop 串行消费写队列
func (sc *SafeConn) writeLoop() {
	// writeLoop 结束后，通过 close(sc.closed) 通知其他 goroutine 连接已关闭
	defer func() {
		// 确保 conn.Close() 在所有写操作完成后执行
		_ = sc.conn.Close()
		close(sc.closed)
	}()
	for fn := range sc.writeCh {
		if err := fn(); err != nil {
			log.Printf("SafeConn write error: %v", err)
			sc.closeErr = err
			// 发生写入错误时，不再接收新的写入任务，并等待循环自然结束
			return
		}
	}
}

// pingLoop 定期发送 Ping 消息以保持连接
func (sc *SafeConn) pingLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// 将 Ping 操作也放入写队列，以保证所有写操作是串行的
			pingFunc := func() error {
				if err := sc.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
					return err
				}
				return sc.WriteMessage(websocket.PingMessage, nil)
			}
			select {
			case sc.writeCh <- pingFunc:
				// Ping 已入队
			case <-sc.closed:
				// 连接已关闭，停止发送 Ping
				log.Println("Connection closed, stopping ping loop.")
				return
			}
		case <-sc.closed:
			// 连接已关闭，停止发送 Ping
			log.Println("Connection closed, stopping ping loop.")
			return
		}
	}
}

// ReadJSON 实现 domain.WebSocketConn 接口
func (sc *SafeConn) ReadJSON(v any) error {
	return sc.conn.ReadJSON(v)
}

// ReadMessage 实现 domain.WebSocketConn 接口
func (sc *SafeConn) ReadMessage() (messageType int, p []byte, err error) {
	return sc.conn.ReadMessage()
}

// WriteJSON 实现 domain.WebSocketConn 接口
func (sc *SafeConn) WriteJSON(v any) error {
	return sc.queueWrite(func() error {
		if err := sc.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
			return err
		}
		return sc.conn.WriteJSON(v)
	})
}

// WriteMessage 实现 domain.WebSocketConn 接口
func (sc *SafeConn) WriteMessage(messageType int, data []byte) error {
	return sc.queueWrite(func() error {
		if err := sc.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
			return err
		}
		return sc.conn.WriteMessage(messageType, data)
	})
}

// SetReadDeadline 实现 domain.WebSocketConn 接口
func (sc *SafeConn) SetReadDeadline(t time.Time) error {
	return sc.conn.SetReadDeadline(t)
}

// SetWriteDeadline 实现 domain.WebSocketConn 接口
func (sc *SafeConn) SetWriteDeadline(t time.Time) error {
	return sc.conn.SetWriteDeadline(t)
}

// SetPongHandler 实现 domain.WebSocketConn 接口
func (sc *SafeConn) SetPongHandler(h func(string) error) {
	sc.conn.SetPongHandler(h)
}

// queueWrite 是一个辅助函数，用于将写操作放入队列
func (sc *SafeConn) queueWrite(fn func() error) error {
	select {
	case sc.writeCh <- fn:
		return nil
	case <-sc.closed:
		return errors.New("connection already closed")
	}
}

// Close 实现 domain.WebSocketConn 接口
func (sc *SafeConn) Close() error {
	sc.once.Do(func() {
		// 关闭写通道，停止写循环
		close(sc.writeCh)
	})
	<-sc.closed // 等待 writeLoop 结束并关闭底层连接
	return sc.closeErr
}
