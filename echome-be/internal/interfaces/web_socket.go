package interfaces

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/internal/response"
	"github.com/labstack/echo/v4"
)

// WebSocketHandlers handles WebSocket connections
type WebSocketHandlers struct {
	webRTCService       domain.WebRTCService
	aiService           domain.AIService
	conversationService domain.ConversationService
}

// NewWebSocketHandlers creates new WebSocket handlers
func NewWebSocketHandlers(webRTCService domain.WebRTCService, aiService domain.AIService, conversationService domain.ConversationService) *WebSocketHandlers {
	return &WebSocketHandlers{
		webRTCService:       webRTCService,
		aiService:           aiService,
		conversationService: conversationService,
	}
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandlers) RegisterRoutes(e *echo.Echo) {
	e.GET("/ws/asr", h.HandleASRWebSocket)
	e.GET("/ws/tts", h.HandleTTSWebSocket)
	e.GET("/ws/webrtc/:sessionId/:userId", h.HandleWebRTCWebSocket)
	e.GET("/ws/voice-conversation/:characterId", h.HandleVoiceConversationWebSocket)
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
	defer ws.Close()

	// Use AI service to handle ASR WebSocket connection
	if err := h.aiService.HandleASR(c.Request().Context(), ws); err != nil {
		log.Printf("ASR WebSocket error: %v", err)
		return err
	}

	log.Printf("ASR WebSocket connection closed")
	return nil
}

// HandleTTSWebSocket handles TTS WebSocket connection
// @Summary 文本转语音WebSocket连接
// @Description 建立文本转语音的WebSocket连接，用于实时文本转语音
// @Param text path string true "文本"
// @Tags websocket
// @Success 101
// @Router /ws/tts [get]
func (h *WebSocketHandlers) HandleTTSWebSocket(c echo.Context) error {
	log.Printf("TTS WebSocket connection requested")

	// Upgrade HTTP connection to WebSocket
	ws, err := upgradeToWebSocket(c)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return err
	}
	defer ws.Close()
	// Use AI service to handle TTS WebSocket connection
	if err := h.aiService.HandleTTS(c.Request().Context(), ws); err != nil {
		log.Printf("TTS WebSocket error: %v", err)
		return err
	}

	log.Printf("TTS WebSocket connection closed")
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

// HandleVoiceConversationWebSocket handles text input and returns AI voice message via WebSocket
// @Summary 单用户语音对话WebSocket连接
// @Description 建立WebSocket连接，用户发送文本，返回AI生成的语音消息（单用户模式，无会话管理）
// @Tags websocket
// @Param characterId path string true "角色ID"
// @Param language query string false "语言" default(zh)
// @Success 101
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ws/voice-conversation/{characterId} [get]
func (h *WebSocketHandlers) HandleVoiceConversationWebSocket(c echo.Context) error {
	log.Printf("Voice conversation WebSocket connection requested")

	// Parse path parameters - 使用正确的参数名
	characterId := c.Param("characterId")
	if characterId == "" {
		log.Printf("Missing character ID")
		return response.BadRequest(c, "Character ID is required")
	}

	// Parse query parameters
	language := c.QueryParam("language")
	if language == "" {
		language = "zh"
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgradeToWebSocket(c)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return err
	}
	defer ws.Close()

	// Send connection established message
	if err := ws.WriteJSON(map[string]any{
		"type":        "connection_established",
		"language":    language,
		"characterId": characterId,
		"timestamp":   time.Now(),
	}); err != nil {
		log.Printf("Failed to send connection established message: %v", err)
		return err
	}

	// 验证characterId格式并解析为UUID
	characterUUID, err := uuid.Parse(characterId)
	if err != nil {
		log.Printf("Invalid characterId format: %v", err)
		return response.BadRequest(c, "Invalid character ID format")
	}
	
	// 检查是否为nil UUID
	if characterUUID == uuid.Nil {
		log.Printf("Nil character ID provided")
		return response.BadRequest(c, "Character ID cannot be nil")
	}

	// Create simplified voice conversation request
	voiceConvReq := &domain.VoiceConversationRequest{
		WebSocketConn: ws,
		CharacterID:   characterUUID,
		Language:      language,
	}

	// Start voice conversation using conversation service
	if err := h.conversationService.StartVoiceConversation(c.Request().Context(), voiceConvReq); err != nil {
		log.Printf("Voice conversation error: %v", err)
		// Send error message to client before closing
		_ = ws.WriteJSON(map[string]string{
			"type":    "error",
			"message": "Failed to start voice conversation: " + err.Error(),
		})
		return err
	}
	return nil
}

// upgradeToWebSocket upgrades an HTTP connection to a WebSocket connection
func upgradeToWebSocket(c echo.Context) (*websocket.Conn, error) {
	// 配置 Upgrader
	upgrader := websocket.Upgrader{
		// 配置缓冲区大小，适合音频流
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		// 设置握手超时
		HandshakeTimeout: 10 * time.Second,
		// 允许所有来源的WebSocket连接，生产环境应该根据需要限制
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 确保响应未提交
	if c.Response().Committed {
		return nil, fmt.Errorf("响应已提交，无法升级到WebSocket")
	}

	// 升级连接
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return nil, fmt.Errorf("WebSocket升级失败: %w", err)
	}

	// 支持上下文取消
	go func() {
		<-c.Request().Context().Done()
		// 优雅关闭WebSocket连接，发送关闭帧
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		// 给对方一些时间处理关闭帧
		time.Sleep(500 * time.Millisecond)
		conn.Close()
	}()

	return conn, nil
}
