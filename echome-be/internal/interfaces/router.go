package interfaces

import (
	"github.com/justin/echome-be/internal/domain"
	"github.com/labstack/echo/v4"
)

type Router struct {
	characterHandlers *CharacterHandlers
	webSocketHandlers *WebSocketHandlers
}

// NewRouter 创建路由
func NewRouter(
	characterService domain.CharacterService,
	webRTCService domain.WebRTCService,
	aiService domain.AIService,
	conversationService domain.ConversationService,
) *Router {
	return &Router{
		characterHandlers: NewCharacterHandlers(characterService),
		webSocketHandlers: NewWebSocketHandlers(webRTCService, aiService, conversationService),
	}
}

// RegisterAllRoutes 注册所有路由
func (r *Router) RegisterAllRoutes(e *echo.Echo) {
	// 注册角色路由
	r.characterHandlers.RegisterRoutes(e)

	// 注册 WebSocket 路由
	r.webSocketHandlers.RegisterRoutes(e)
}
