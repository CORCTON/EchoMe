package domain

// WSWriter 定义统一的WebSocket写接口
type WSWriter interface {
	WriteJSON(v any) error
}
