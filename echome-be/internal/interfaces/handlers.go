package interfaces

import (
	"github.com/justin/echome-be/internal/domain"
	"github.com/labstack/echo/v4"
)

type Handlers struct {
	router *Router
}

// NewHandlers
func NewHandlers(characterService domain.CharacterService, webRTCService domain.WebRTCService, aiService domain.AIService, conversationService domain.ConversationService) *Handlers {
	router := NewRouter(characterService, webRTCService, aiService, conversationService)
	return &Handlers{
		router: router,
	}
}

// RegisterRoutes
func (h *Handlers) RegisterRoutes(e *echo.Echo) {
	h.router.RegisterAllRoutes(e)
}

// GetRouter
func (h *Handlers) GetRouter() *Router {
	return h.router
}
