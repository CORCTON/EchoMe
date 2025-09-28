package handler

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain/ai"
	"github.com/justin/echome-be/internal/domain/conversation"
	"github.com/justin/echome-be/internal/infra/ws"
	"github.com/labstack/echo/v4"
)

type WebSocketHandlers struct {
	aiClient            ai.Repo
	conversationService *conversation.ConversationService
}

func NewWebSocketHandlers(aiClient ai.Repo, conversationService *conversation.ConversationService) *WebSocketHandlers {
	return &WebSocketHandlers{
		aiClient:            aiClient,
		conversationService: conversationService,
	}
}

// RegisterRoutes 注册路由
func (h *WebSocketHandlers) RegisterRoutes(e *echo.Echo) {
	e.GET("/ws/asr", h.HandleASRWebSocket)
	e.GET("/ws/voice-conversation", h.HandleVoiceConversationWebSocket)
}

// HandleASRWebSocket handles ASR WebSocket connection
// @Summary 语音识别WebSocket连接
// @Description 建立语音识别的WebSocket连接，用于实时语音转文本
// @Tags websocket
// @Success 101
// @Router /ws/asr [get]
func (h *WebSocketHandlers) HandleASRWebSocket(c echo.Context) error {
	// Upgrade HTTP connection to WebSocket
	ws, err := upgradeToWebSocket(c)
	if err != nil {
		zap.L().Error("Failed to upgrade to WebSocket", zap.Error(err))
		return err
	}
	// Use AI service to handle ASR WebSocket connection
	if err := h.aiClient.HandleASR(c.Request().Context(), ws); err != nil {
		zap.L().Error("ASR WebSocket error", zap.Error(err))
		return err
	}

	return nil
}

// HandleVoiceConversationWebSocket handles voice conversation via WebSocket
// @Summary 语音对话WebSocket连接
// @Description 建立WebSocket连接，用户通过WebSocket消息发送语音或文本，返回AI生成的响应
// @Tags websocket
// @Param characterId query string false "角色ID"
// @Success 101
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ws/voice-conversation [get]
func (h *WebSocketHandlers) HandleVoiceConversationWebSocket(c echo.Context) error {

	// 获取查询参数
	characterID := c.QueryParam("characterId")
	if characterID == "" {
		characterID = uuid.Nil.String()
	}

	// 升级到WebSocket
	ws, err := upgradeToWebSocket(c)
	if err != nil {
		return err
	}

	// 发送连接建立消息
	if err := ws.WriteJSON(map[string]any{
		"type":      "connection_established",
		"timestamp": time.Now(),
	}); err != nil {
		return err
	}

	cid, err := uuid.Parse(characterID)
	if err != nil {
		cid = uuid.Nil
	}
	// 创建语音对话请求
	voiceConvReq := &conversation.VoiceConversationRequest{
		SafeConn:    ws,
		CharacterID: cid,
	}

	// 启动语音对话
	if err := h.conversationService.StartVoiceConversation(c.Request().Context(), voiceConvReq); err != nil {
		_ = ws.WriteJSON(map[string]string{
			"type":    "error",
			"message": "Failed to start voice conversation: " + err.Error(),
		})
		return err
	}
	return nil
}

// 升级HTTP连接到WebSocket
func upgradeToWebSocket(c echo.Context) (*ws.SafeConn, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		HandshakeTimeout: 10 * time.Second,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	if c.Response().Committed {
		return nil, fmt.Errorf("响应已提交，无法升级到WebSocket")
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return nil, fmt.Errorf("WebSocket升级失败: %w", err)
	}

	go func() {
		<-c.Request().Context().Done()
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(500 * time.Millisecond)
	}()

	return ws.NewSafeConn(conn), nil
}
