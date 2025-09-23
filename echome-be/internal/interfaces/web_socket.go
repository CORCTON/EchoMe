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
	e.GET("/ws/voice-conversation/:sessionId/:characterId", h.HandleVoiceConversationWebSocket)
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
// @Tags websocket
// @Param model query string false "TTS模型名称" default(qwen-tts-realtime)
// @Param voice query string false "语音ID" default(Cherry)
// @Param response_format query string false "响应格式" default(pcm)
// @Param sample_rate query int false "采样率" default(24000)
// @Param mode query string false "模式" default(server_commit)
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

	// Parse query parameters for TTS configuration
	model := c.QueryParam("model")
	if model == "" {
		model = "qwen-tts-realtime"
	}

	voice := c.QueryParam("voice")
	if voice == "" {
		voice = "Cherry"
	}

	responseFormat := c.QueryParam("response_format")
	if responseFormat == "" {
		responseFormat = "pcm"
	}

	sampleRate := 24000
	if sr := c.QueryParam("sample_rate"); sr != "" {
		if parsed, err := parseIntParam(sr); err == nil {
			sampleRate = parsed
		}
	}

	mode := c.QueryParam("mode")
	if mode == "" {
		mode = "server_commit"
	}

	// Create TTS configuration
	ttsConfig := domain.TTSConfig{
		Model:          model,
		Voice:          voice,
		ResponseFormat: responseFormat,
		SampleRate:     sampleRate,
		Mode:           mode,
	}

	log.Printf("Starting TTS WebSocket with config: model=%s, voice=%s, format=%s, sample_rate=%d, mode=%s",
		ttsConfig.Model, ttsConfig.Voice, ttsConfig.ResponseFormat, ttsConfig.SampleRate, ttsConfig.Mode)

	// Use AI service to handle TTS WebSocket connection
	if err := h.aiService.HandleTTS(c.Request().Context(), ws, ttsConfig); err != nil {
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
		return response.BadRequest(c,"Invalid session ID")
	}

	userID := c.Param("userId")
	if userID == "" {
		return response.BadRequest(c,"User ID is required")
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

// HandleVoiceConversationWebSocket handles voice conversation WebSocket connection
// @Summary 语音对话WebSocket连接
// @Description 建立语音对话的WebSocket连接，整合ASR、AI和TTS的完整流程
// @Tags websocket
// @Param sessionId path string true "会话ID"
// @Param characterId path string true "角色ID"
// @Param userId query string false "用户ID"
// @Param language query string false "语言" default(zh-CN)
// @Success 101
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /ws/voice-conversation/{sessionId}/{characterId} [get]
func (h *WebSocketHandlers) HandleVoiceConversationWebSocket(c echo.Context) error {
	log.Printf("Voice conversation WebSocket connection requested")

	// Parse path parameters
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		log.Printf("Invalid session ID: %v", err)
		return response.BadRequest(c,"Invalid session ID")
	}

	characterID, err := uuid.Parse(c.Param("characterId"))
	if err != nil {
		log.Printf("Invalid character ID: %v", err)
		return response.BadRequest(c,"Invalid character ID")
	}

	// Parse query parameters
	userID := c.QueryParam("userId")
	if userID == "" {
		userID = "anonymous" // Default user ID if not provided
	}

	language := c.QueryParam("language")
	if language == "" {
		language = "zh-CN"
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgradeToWebSocket(c)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return err
	}

	log.Printf("Starting voice conversation: session=%s, character=%s, user=%s, language=%s",
		sessionID, characterID, userID, language)

	// Create voice conversation request
	voiceConvReq := &domain.VoiceConversationRequest{
		SessionID:     sessionID,
		CharacterID:   characterID,
		WebSocketConn: ws,
		UserID:        userID,
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
		ws.Close()
		return err
	}

	log.Printf("Voice conversation completed for session %s", sessionID)
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

// parseIntParam parses an integer parameter from string
func parseIntParam(param string) (int, error) {
	var result int
	if _, err := fmt.Sscanf(param, "%d", &result); err != nil {
		return 0, err
	}
	return result, nil
}
