package handler

import (
	"github.com/justin/echome-be/internal/domain/ai"
	"github.com/justin/echome-be/internal/domain/character"
	"github.com/justin/echome-be/internal/domain/conversation"
	"github.com/labstack/echo/v4"
)

type Handlers struct {
	router *Router
}

// NewHandlers
func NewHandlers(characterService *character.CharacterService, aiService ai.Repo, conversationService *conversation.ConversationService) *Handlers {
	router := NewRouter(characterService, aiService, conversationService)
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
