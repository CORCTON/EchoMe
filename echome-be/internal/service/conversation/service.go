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
	"golang.org/x/sync/errgroup"
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
		character, err = s.characterService.GetCharacterByID(ctx,req.CharacterID)
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

	characterContext := ""
	var voiceProfile *domain.VoiceProfile
	if character != nil {
		characterContext = character.Description
		voiceProfile = character.VoiceConfig
	}

	// Channel for LLM text chunks
	llmTextChan := make(chan string, 100) // Buffered channel

	// Configure TTS
	ttsConfig := aliyun.DefaultTTSConfig()
	if voiceProfile != nil {
		ttsConfig.Voice = voiceProfile.Voice
	}

	g, conversationCtx := errgroup.WithContext(ctx)

	// Goroutine 1: Handle TTS streaming
	g.Go(func() error {
		// Note: We are passing the conversationCtx which can be canceled by the errgroup.
		return s.aiService.HandleCosyVoiceTTS(conversationCtx, sc, llmTextChan, ttsConfig)
	})

	// Goroutine 2: Generate LLM response and send to channel
	g.Go(func() error {
		defer close(llmTextChan) // Close channel when LLM stream ends

		onChunk := func(chunk string) error {
			if chunk != "" {
				fullResponse.WriteString(chunk)
				// Send text chunk to client for display
				if err := sc.WriteJSON(map[string]any{
					"type":      "stream_chunk",
					"content":   chunk,
					"timestamp": time.Now(),
				}); err != nil {
					return err // Stop if we can't write to client
				}

				// Send chunk to TTS channel
				select {
				case llmTextChan <- chunk:
				case <-conversationCtx.Done():
					return conversationCtx.Err()
				}
			}
			return nil
		}

		return s.aiService.GenerateStreamResponse(conversationCtx, userInput, characterContext, conversationHistory, onChunk)
	})

	// Wait for both goroutines to finish
	if err := g.Wait(); err != nil {
		// Don't return error on context cancellation, as it's an expected way to stop.
		if err != context.Canceled {
			s.logger.Printf("流式对话处理失败: %v", err)
			_ = sc.WriteJSON(map[string]any{
				"type":    "error",
				"message": "流式响应处理失败: " + err.Error(),
			})
			return err
		}
	}

	// Send stream end message
	_ = sc.WriteJSON(map[string]any{
		"type":      "stream_end",
		"response":  fullResponse.String(),
		"timestamp": time.Now(),
	})
	return nil
}
