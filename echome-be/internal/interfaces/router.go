package interfaces

import (
	"github.com/justin/echome-be/internal/domain"
	"github.com/labstack/echo/v4"
)

// Router manages all HTTP route registrations
type Router struct {
	characterHandlers *CharacterHandlers
	sessionHandlers   *SessionHandlers
	webSocketHandlers *WebSocketHandlers
}

// NewRouter creates a new router with all required handlers
func NewRouter(
	characterService domain.CharacterService,
	sessionService domain.SessionService,
	webRTCService domain.WebRTCService,
	aiService domain.AIService,
	conversationService domain.ConversationService,
) *Router {
	return &Router{
		characterHandlers: NewCharacterHandlers(characterService),
		sessionHandlers:   NewSessionHandlers(sessionService),
		webSocketHandlers: NewWebSocketHandlers(webRTCService, aiService, conversationService),
	}
}

// RegisterAllRoutes registers all API routes
func (r *Router) RegisterAllRoutes(e *echo.Echo) {
	// Register character routes
	r.characterHandlers.RegisterRoutes(e)

	// Register session routes
	r.sessionHandlers.RegisterRoutes(e)

	// Register WebSocket routes
	r.webSocketHandlers.RegisterRoutes(e)
}
