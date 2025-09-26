package conversation

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"

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
}

// NewConversationService 创建会话服务
func NewConversationService(
	aiService domain.AIService,
	characterService domain.CharacterService,
) *ConversationService {
	return &ConversationService{
		aiService:        aiService,
		characterService: characterService,
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
			zap.L().Warn("获取角色失败", zap.Error(err))
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
			zap.L().Warn("读取WebSocket消息失败", zap.Error(err))
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				zap.L().Info("读取超时，客户端无响应，关闭连接")
				return nil
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				zap.L().Warn("客户端意外关闭连接", zap.Error(err))
			}
			return nil
		}
		if messageType != websocket.TextMessage {
			zap.L().Warn("预期文本消息，收到类型不匹配", zap.Int("message_type", messageType))
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
				zap.L().Debug("使用提供的对话历史", zap.Int("message_count", len(conversationHistory)))
			} else {
				zap.L().Warn("解析JSON消息失败", zap.Error(err))
				continue
			}
		}

		characterContext := ""
		if character != nil && character.Prompt != "" {
			characterContext = character.Prompt
		}

		if structuredMessage.Stream {
			if err := s.handleStreamingConversation(conversationCtx, sc, userInput, character, conversationHistory); err != nil {
				zap.L().Error("流式对话处理失败", zap.Error(err))
				_ = sc.WriteJSON(map[string]any{
					"type":    "error",
					"message": "流式响应处理失败",
				})
				continue
			}
		} else {
			response, err := s.aiService.GenerateResponse(conversationCtx, userInput, characterContext, conversationHistory)
			if err != nil {
				zap.L().Error("生成AI响应失败", zap.Error(err))
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

	// Channel for LLM text chunks
	llmTextChan := make(chan string, 100) // Buffered channel

	// 根据角色是否开启复刻决定是否使用复刻音色
	var ttsConfig domain.TTSConfig
	if character != nil && character.Flag && character.Voice!=""{
		ttsConfig = aliyun.DefaultVoiceCloneTTSConfig()
		ttsConfig.Voice = character.Voice
	} else {
		ttsConfig = aliyun.DefaultTTSConfig()
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

		return s.aiService.GenerateStreamResponse(conversationCtx, userInput, 	character.Prompt, conversationHistory, onChunk)
	})

	// Wait for both goroutines to finish
	if err := g.Wait(); err != nil {
		// Don't return error on context cancellation, as it's an expected way to stop.
		if err != context.Canceled {
			zap.L().Error("流式对话处理失败", zap.Error(err))
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
