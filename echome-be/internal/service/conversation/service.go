package conversation

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/client/aliyun"
	"github.com/justin/echome-be/internal/domain"
)

// Constants 定义常量
const (
	TTSMinLength   = 50
	TTSPunctuation = "。！？.!?"
)

// ConversationService 会话服务实现
type ConversationService struct {
	aiService        domain.AIService
	characterService domain.CharacterService
	logger           *log.Logger
}

// NewConversationService 创建会话服务
func NewConversationService(
	aiService domain.AIService,
	characterService domain.CharacterService,
) *ConversationService {
	return &ConversationService{
		aiService:        aiService,
		characterService: characterService,
		logger:           log.Default(),
	}
}

var _ domain.ConversationService = (*ConversationService)(nil) // 确保 ConversationService 实现了接口

// StartVoiceConversation 开始语音会话
func (s *ConversationService) StartVoiceConversation(ctx context.Context, req *domain.VoiceConversationRequest) error {
	var character *domain.Character
	var err error

	if req.CharacterID != uuid.Nil {
		character, err = s.characterService.GetCharacterByID(req.CharacterID)
		if err != nil {
			s.logger.Printf("获取角色失败: %v", err)
			character = nil
		}
	}
	return s.handleSimpleVoiceConversationFlow(ctx, req.SafeConn, character)
}

// handleSimpleVoiceConversationFlow 处理语音对话流程
func (s *ConversationService) handleSimpleVoiceConversationFlow(ctx context.Context, sc domain.WebSocketConn, character *domain.Character) error {
	conversationCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		messageType, message, err := sc.ReadMessage()
		if err != nil {
			s.logger.Printf("读取 WebSocket 消息失败: %v", err)
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				s.logger.Println("读取超时: 客户端无响应，关闭连接")
				return nil
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Printf("客户端意外关闭连接: %v", err)
			}
			return nil
		}
		if messageType != websocket.TextMessage {
			s.logger.Printf("预期文本消息，收到类型: %d", messageType)
			continue
		}

		var structuredMessage struct {
			Text     string                  `json:"text"`
			Messages []domain.ContextMessage `json:"messages,omitempty"`
			Stream   bool                    `json:"stream,omitempty"`
		}

		userInput := string(message)
		conversationHistory := []map[string]string{}

		if err := json.Unmarshal(message, &structuredMessage); err == nil {
			if structuredMessage.Text != "" {
				userInput = structuredMessage.Text
			}
			if len(structuredMessage.Messages) > 0 {
				conversationHistory = make([]map[string]string, 0, len(structuredMessage.Messages))
				for _, msg := range structuredMessage.Messages {
					conversationHistory = append(conversationHistory, map[string]string{
						"role":    msg.Role,
						"content": msg.Content,
					})
				}
				s.logger.Printf("使用提供的对话历史，消息数: %d", len(conversationHistory))
			}
		} else {
			s.logger.Printf("解析 JSON 消息失败: %v", err)
			continue
		}

		characterContext := ""
		if character != nil {
			characterContext = character.Description
		}

		if structuredMessage.Stream {
			if err := s.handleStreamingConversation(conversationCtx, sc, userInput, character, conversationHistory); err != nil {
				s.logger.Printf("流式对话处理失败: %v", err)
				_ = sc.WriteJSON(map[string]any{
					"type":    "error",
					"message": "流式响应处理失败",
				})
				continue
			}
		} else {
			response, err := s.aiService.GenerateResponse(conversationCtx, userInput, characterContext, conversationHistory)
			if err != nil {
				s.logger.Printf("生成 AI 响应失败: %v", err)
				return WrapError(ErrCodeAIGenerationFailed, "生成AI响应失败", err)
			}

			_ = sc.WriteJSON(map[string]any{
				"type":      "text_response",
				"response":  response,
				"timestamp": time.Now(),
			})
		}
	}
}

// handleStreamingConversation 处理流式对话
func (s *ConversationService) handleStreamingConversation(
	ctx context.Context,
	sc domain.WebSocketConn,
	userInput string,
	character *domain.Character,
	conversationHistory []map[string]string,
) error {
	_ = sc.WriteJSON(map[string]any{"type": "stream_start", "timestamp": time.Now()})

	var fullResponse strings.Builder
	var ttsBuffer strings.Builder

	characterContext := ""
	var voiceProfile *domain.VoiceProfile
	if character != nil {
		characterContext = character.Description
		voiceProfile = character.VoiceConfig
	}

	ttsChan := make(chan string)
	go s.processTTSQueue(ctx, sc, voiceProfile, ttsChan)

	onChunk := func(chunk string) error {
		fullResponse.WriteString(chunk)
		_ = sc.WriteJSON(map[string]any{
			"type":      "stream_chunk",
			"content":   chunk,
			"timestamp": time.Now(),
		})

		ttsBuffer.WriteString(chunk)
		text := ttsBuffer.String()
		shouldTTS := len(text) >= TTSMinLength || (len(text) > 0 && strings.ContainsAny(text[len(text)-1:], TTSPunctuation))

		if shouldTTS {
			ttsChan <- text
			ttsBuffer.Reset()
		}
		return nil
	}

	if err := s.aiService.GenerateStreamResponse(ctx, userInput, characterContext, conversationHistory, onChunk); err != nil {
		close(ttsChan)
		return WrapError(ErrCodeAIGenerationFailed, "生成流式AI响应失败", err)
	}

	if ttsBuffer.Len() > 0 {
		ttsChan <- ttsBuffer.String()
	}

	close(ttsChan)
	_ = sc.WriteJSON(map[string]any{
		"type":      "stream_end",
		"response":  fullResponse.String(),
		"timestamp": time.Now(),
	})
	return nil
}

// processTTSQueue 处理 TTS 队列
func (s *ConversationService) processTTSQueue(
	ctx context.Context,
	sc domain.WebSocketConn,
	voiceProfile *domain.VoiceProfile,
	ttsChan <-chan string,
) {
	for toSpeak := range ttsChan {
		cfg := aliyun.DefaultTTSConfig()
		if voiceProfile != nil {
			cfg.Voice = voiceProfile.Voice
		}
		// 假设 HandleTTS 需要 text 参数
		if err := s.aiService.HandleTTS(ctx, sc, toSpeak, cfg); err != nil {
			s.logger.Printf("TTS处理失败 (文本: %q): %v", toSpeak, err)
			_ = sc.WriteJSON(map[string]any{
				"type":    "tts_error",
				"message": "TTS 处理失败",
			})
		}
	}
}
