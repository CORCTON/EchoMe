package interfaces

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/internal/infra"
	"github.com/justin/echome-be/internal/response"
	"github.com/labstack/echo/v4"
)

type WebSocketHandlers struct {
	webRTCService       domain.WebRTCService
	aiService           domain.AIService
	conversationService domain.ConversationService
}

func NewWebSocketHandlers(webRTCService domain.WebRTCService, aiService domain.AIService, conversationService domain.ConversationService) *WebSocketHandlers {
	return &WebSocketHandlers{
		webRTCService:       webRTCService,
		aiService:           aiService,
		conversationService: conversationService,
	}
}

// RegisterRoutes 注册路由
func (h *WebSocketHandlers) RegisterRoutes(e *echo.Echo) {
	e.GET("/ws/asr", h.HandleASRWebSocket)
	e.GET("/ws/webrtc/:sessionId/:userId", h.HandleWebRTCWebSocket)
	e.GET("/ws/voice-conversation", h.HandleVoiceConversationWebSocket)
}

// HandleASRWebSocket handles ASR WebSocket connection
// @Summary 语音识别WebSocket连接
// @Description 建立语音识别的WebSocket连接，用于实时语音转文本
// @Tags websocket
// @Success 101
// @Router /ws/asr [get]
func (h *WebSocketHandlers) HandleASRWebSocket(c echo.Context) error {
	log.Printf("ASR WebSocket connection requested")

	// Upgrade HTTP connection to WebSocket
	ws, err := upgradeToWebSocket(c)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return err
	}
	// Use AI service to handle ASR WebSocket connection
	if err := h.aiService.HandleASR(c.Request().Context(), ws); err != nil {
		log.Printf("ASR WebSocket error: %v", err)
		return err
	}

	log.Printf("ASR WebSocket connection closed")
	return nil
}

// HandleWebRTCWebSocket handles WebRTC signaling WebSocket connection
// @Summary WebRTC信令WebSocket连接
// @Description 建立WebRTC信令的WebSocket连接，用于点对点通信
// @Tags websocket
// @Param sessionId path string true "会话ID"
// @Param userId path string true "用户ID"
// @Success 101
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ws/webrtc/{sessionId}/{userId} [get]
func (h *WebSocketHandlers) HandleWebRTCWebSocket(c echo.Context) error {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		return response.BadRequest(c, "Invalid session ID")
	}

	userID := c.Param("userId")
	if userID == "" {
		return response.BadRequest(c, "User ID is required")
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgradeToWebSocket(c)
	if err != nil {
		return err
	}

	// Create a WebRTC peer connection
	connection, err := h.webRTCService.CreatePeerConnection(sessionID, userID)
	if err != nil {
		ws.Close()
		return response.InternalError(c, "Failed to create WebRTC peer connection", err.Error())
	}

	// Store the WebSocket connection
	connection.SocketConn = ws

	// Send connection ID to client
	if err := ws.WriteJSON(map[string]interface{}{
		"type":         "connection-established",
		"connectionId": connection.ID,
	}); err != nil {
		if closeErr := h.webRTCService.ClosePeerConnection(connection.ID); closeErr != nil {
			log.Printf("Failed to close WebRTC peer connection: %v", closeErr)
		}
		return err
	}

	// Handle WebRTC signaling
	for {
		var signal domain.SignalMessage
		if err := ws.ReadJSON(&signal); err != nil {
			// Connection closed or error occurred
			break
		}

		// Process the signal
		if err := h.webRTCService.HandleSignal(connection.ID, signal); err != nil {
			// Send error to client
			if err := ws.WriteJSON(map[string]string{"error": err.Error()}); err != nil {
				log.Printf("Failed to send error to client: %v", err)
			}
		}
	}

	// Clean up connection
	if err := h.webRTCService.ClosePeerConnection(connection.ID); err != nil {
		log.Printf("Failed to close WebRTC peer connection: %v", err)
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
	log.Println("Voice conversation WebSocket connection requested")

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

	// 创建语音对话请求
	voiceConvReq := &domain.VoiceConversationRequest{
			SafeConn: ws,
			CharacterID: uuid.MustParse(characterID),
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
func upgradeToWebSocket(c echo.Context) (*infra.SafeConn, error) {
	upgrader := websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
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

	return infra.NewSafeConn(conn), nil
}
