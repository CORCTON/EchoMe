package interfaces

import (
	"github.com/justin/echome-be/internal/domain"
	"github.com/labstack/echo/v4"
)

type Handlers struct {
	router     *Router
	DBAdapter  domain.DBAdapter
}

// NewHandlers
func NewHandlers(characterService domain.CharacterService, webRTCService domain.WebRTCService, aiService domain.AIService, conversationService domain.ConversationService, dbAdapter domain.DBAdapter) *Handlers {
	router := NewRouter(characterService, webRTCService, aiService, conversationService)
	return &Handlers{
		router:    router,
		DBAdapter: dbAdapter,
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
