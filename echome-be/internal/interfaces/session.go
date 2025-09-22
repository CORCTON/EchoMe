package interfaces

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/justin/echome-be/internal/domain"
)

// SessionHandlers handles session-related HTTP requests
type SessionHandlers struct {
	sessionService domain.SessionService
}

// NewSessionHandlers creates new session handlers
func NewSessionHandlers(sessionService domain.SessionService) *SessionHandlers {
	return &SessionHandlers{
		sessionService: sessionService,
	}
}

// RegisterRoutes registers session-related routes
func (h *SessionHandlers) RegisterRoutes(e *echo.Echo) {
	e.POST("/api/sessions", h.CreateSession)
	e.GET("/api/sessions", h.GetUserSessions)
	e.GET("/api/sessions/:id", h.GetSessionByID)
	e.GET("/api/sessions/:id/messages", h.GetSessionMessages)
	e.POST("/api/sessions/:id/messages", h.SendMessage)
}

// CreateSession handles POST /api/sessions
// @Summary 创建新会话
// @Description 创建一个新的用户会话
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body object{userId=string,characterId=string} true "会话创建请求"
// @Success 201 {object} domain.Session
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/sessions [post]
func (h *SessionHandlers) CreateSession(c echo.Context) error {
	var request struct {
		UserID      string    `json:"userId"`
		CharacterID uuid.UUID `json:"characterId"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	session, err := h.sessionService.CreateSession(request.UserID, request.CharacterID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, session)
}

// GetUserSessions handles GET /api/sessions
// @Summary 获取用户会话列表
// @Description 获取指定用户的所有会话
// @Tags sessions
// @Accept json
// @Produce json
// @Param userId query string true "用户ID"
// @Success 200 {array} domain.Session
// @Failure 500 {object} map[string]string
// @Router /api/sessions [get]
func (h *SessionHandlers) GetUserSessions(c echo.Context) error {
	userID := c.QueryParam("userId")
	sessions, err := h.sessionService.GetUserSessions(userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, sessions)
}

// GetSessionByID handles GET /api/sessions/:id
// @Summary 获取会话详情
// @Description 根据ID获取特定会话的详细信息
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Success 200 {object} domain.Session
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/sessions/{id} [get]
func (h *SessionHandlers) GetSessionByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid session ID"})
	}

	session, err := h.sessionService.GetSessionByID(id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Session not found"})
	}

	return c.JSON(200, session)
}

// GetSessionMessages handles GET /api/sessions/:id/messages
// @Summary 获取会话消息
// @Description 获取特定会话的所有消息
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Success 200 {array} domain.Message
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/sessions/{id}/messages [get]
func (h *SessionHandlers) GetSessionMessages(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid session ID"})
	}

	messages, err := h.sessionService.GetSessionMessages(id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, messages)
}

// SendMessage handles POST /api/sessions/:id/messages
// @Summary 发送消息
// @Description 向特定会话发送消息
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Param request body object{content=string,sender=string} true "消息内容"
// @Success 201 {object} domain.Message
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/sessions/{id}/messages [post]
func (h *SessionHandlers) SendMessage(c echo.Context) error {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid session ID"})
	}

	var request struct {
		Content string `json:"content"`
		Sender  string `json:"sender"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	message, err := h.sessionService.SendMessage(sessionID, request.Content, request.Sender)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, message)
}