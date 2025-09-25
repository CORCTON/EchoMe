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

// GetCharacterVoiceConfig 获取角色的语音配置
func (s *ConversationService) GetCharacterVoiceConfig(characterID uuid.UUID) (*domain.VoiceConfig, error) {
	// 获取角色信息
	character, err := s.characterService.GetCharacterByID(characterID)
	if err != nil {
		return nil, WrapError(ErrCodeCharacterNotFound, "获取角色失败", err)
	}

	// 创建默认的ASR和TTS配置
	asrConfig := aliyun.DefaultASRConfig()
	ttsConfig := aliyun.DefaultTTSConfig()

	// 如果角色有自定义的语音配置，可以在这里覆盖默认配置
	// 例如：
	// if character.VoiceConfig != nil {
	//     asrConfig = character.VoiceConfig.ASRConfig
	//     ttsConfig = character.VoiceConfig.TTSConfig
	// }

	// 创建并返回语音配置
	return &domain.VoiceConfig{
		Character: character,
		ASRConfig: asrConfig,
		TTSConfig: ttsConfig,
		Language:  "zh",
	}, nil
}

// ProcessTextMessage 处理文本消息并返回AI响应
func (s *ConversationService) ProcessTextMessage(ctx context.Context, req *domain.TextMessageRequest) (*domain.TextMessageResponse, error) {
	// 设置对话历史长度限制
	const maxContextLength = 20 // 最大上下文消息数量

	// 验证上下文长度
	if len(req.Messages) > maxContextLength {
		return nil, WrapError(ErrCodeInvalidInput, "上下文消息数量超过限制", nil)
	}

	// 获取角色信息
	var character *domain.Character
	var err error

	// 如果请求中提供了角色ID，则使用该角色
	if req.CharacterID != "" {
		characterID, parseErr := uuid.Parse(req.CharacterID)
		if parseErr != nil {
			return nil, WrapError(ErrCodeInvalidInput, "无效的角色ID", parseErr)
		}
		character, err = s.characterService.GetCharacterByID(characterID)
		if err != nil {
			return nil, WrapError(ErrCodeCharacterNotFound, "获取角色失败", err)
		}
	} else {
		// 否则使用默认角色
		defaultCharacterID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
		character, err = s.characterService.GetCharacterByID(defaultCharacterID)
		if err != nil {
			return nil, WrapError(ErrCodeCharacterNotFound, "获取默认角色失败", err)
		}
	}

	// 将前端提供的上下文消息转换为AI服务需要的格式
	conversationHistory := make([]map[string]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		conversationHistory = append(conversationHistory, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	// 使用AI服务生成响应，传入前端提供的对话历史
	response, err := s.aiService.GenerateResponse(ctx, req.UserInput, character.Description, conversationHistory)
	if err != nil {
		return nil, WrapError(ErrCodeAIGenerationFailed, "生成AI响应失败", err)
	}

	// 创建响应对象
	return &domain.TextMessageResponse{
		Response:  response,
		MessageID: uuid.New(),
		Timestamp: time.Now(),
	}, nil
}

// StartVoiceConversation 开始语音会话
func (s *ConversationService) StartVoiceConversation(ctx context.Context, req *domain.VoiceConversationRequest) error {
	// 角色ID可以为空，将在handleSimpleVoiceConversationFlow中从JSON消息获取
	var character *domain.Character
	var err error

	// 如果提供了角色ID，先尝试获取角色
	if req.CharacterID != uuid.Nil {
		character, err = s.characterService.GetCharacterByID(req.CharacterID)
		if err != nil {
			// 如果角色不存在，创建一个默认角色
			log.Printf("Character not found, using default character: %v", err)
			character = &domain.Character{
				ID:          req.CharacterID,
				Name:        "默认角色",
				Description: "你是一个友好的AI助手，乐于回答用户的问题。",
			}
		}
	} else {
		// 未提供角色ID，使用临时ID创建默认角色
		character = &domain.Character{
			ID:          uuid.New(),
			Name:        "临时默认角色",
			Description: "你是一个友好的AI助手，乐于回答用户的问题。",
		}
	}

	return s.handleSimpleVoiceConversationFlow(ctx, req.SafeConn, character, req.Language)
}

// handleSimpleVoiceConversationFlow 处理语音对话流程
func (s *ConversationService) handleSimpleVoiceConversationFlow(ctx context.Context, sc domain.WebSocketConn, character *domain.Character, language string) error {
	log.Println("Starting voice conversation flow with character:", character.ID)

	conversationCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		messageType, message, err := sc.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)

			// 检查是否是“超时”错误，这表示客户端已失联
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Println("Read timeout: client is unresponsive. Closing connection.")
				return nil
			}

			// 检查是否是客户端主动关闭的错误
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Client closed connection unexpectedly: %v", err)
			}
			// 对于所有其他错误（包括正常关闭），退出循环
			return nil 
		}
		if messageType != websocket.TextMessage {
			log.Printf("Expected text message but received %d", messageType)
			continue
		}

		// 处理输入消息
		var structuredMessage struct {
			Text        string                  `json:"text"`
			CharacterID string                  `json:"character_id,omitempty"`
			Messages    []domain.ContextMessage `json:"messages,omitempty"`
			Stream      bool                    `json:"stream,omitempty"` // 是否启用流式响应
		}

		userInput := string(message)
		conversationHistory := []map[string]string{}
		enableStream := false

		// 尝试解析JSON结构
		if err := json.Unmarshal(message, &structuredMessage); err == nil {
			// 成功解析为结构化消息
			if structuredMessage.Text != "" {
				userInput = structuredMessage.Text
			}

			// 处理角色ID
			if structuredMessage.CharacterID != "" {
				log.Printf("Received character ID: %s", structuredMessage.CharacterID)
				characterID, parseErr := uuid.Parse(structuredMessage.CharacterID)
				if parseErr == nil {
					// 尝试获取新的角色
					newCharacter, charErr := s.characterService.GetCharacterByID(characterID)
					if charErr == nil {
						character = newCharacter
						log.Printf("Updated conversation character to: %s", character.Name)
					} else {
						log.Printf("Failed to get character by ID, using current character: %v", charErr)
					}
				} else {
					log.Printf("Invalid character ID format: %v", parseErr)
				}
			}

			// 转换上下文消息格式
			if len(structuredMessage.Messages) > 0 {
				conversationHistory = make([]map[string]string, 0, len(structuredMessage.Messages))
				for _, msg := range structuredMessage.Messages {
					conversationHistory = append(conversationHistory, map[string]string{
						"role":    msg.Role,
						"content": msg.Content,
					})
				}
				log.Printf("Using provided conversation history with %d messages", len(conversationHistory))
			}

			// 检查是否启用流式响应
			enableStream = structuredMessage.Stream
		} else {
			// 非结构化消息，直接作为用户输入
			log.Printf("Received plain text message")
		}

		log.Printf("Processing user input: %s", userInput)

		characterContext := character.Description

	// 根据是否启用流式响应选择不同的处理方式
	if enableStream {
		if err := s.handleStreamingConversation(conversationCtx, sc, userInput, characterContext, conversationHistory); err != nil {
			sc.WriteJSON(map[string]any{
				"type":    "error",
				"message": "流式响应处理失败",
			})
			continue
		}
	} else {
		response, err := s.aiService.GenerateResponse(conversationCtx, userInput, characterContext, conversationHistory)
		if err != nil {
			return WrapError(ErrCodeAIGenerationFailed, "生成AI响应失败", err)
		}

		sc.WriteJSON(map[string]any{
			"type":      "text_response",
			"response":  response,
			"timestamp": time.Now(),
		})
		go func(text string) {
			ttsCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			if err := s.aiService.TextToSpeech(ttsCtx, text, sc); err != nil {
				sc.WriteJSON(map[string]any{
					"type":    "tts_error",
					"message": err.Error(),
				})
			}
		}(response)
	}
}
}

func (s *ConversationService) handleStreamingConversation(
	ctx context.Context,
	sc domain.WebSocketConn,
	userInput string,
	characterContext string,
	conversationHistory []map[string]string,
) error {
	// 直接用 sc 写，保证不会并发写 conn
	sc.WriteJSON(map[string]any{"type": "stream_start", "timestamp": time.Now()})

	var fullResponse, ttsBuffer strings.Builder
	const ttsMinLen = 50

	onChunk := func(chunk string) error {
		fullResponse.WriteString(chunk)
		sc.WriteJSON(map[string]any{
			"type":      "stream_chunk",
			"content":   chunk,
			"timestamp": time.Now(),
		})

		ttsBuffer.WriteString(chunk)
		text := ttsBuffer.String()
		shouldTTS := len(text) >= ttsMinLen || strings.ContainsAny(text[len(text)-1:], "。！？.!?")
		if shouldTTS {
			toSpeak := text
			ttsBuffer.Reset()
			go func() {
				ttsCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				if err := s.aiService.TextToSpeech(ttsCtx, toSpeak, sc); err != nil {
					sc.WriteJSON(map[string]any{
						"type":    "tts_error",
						"message": err.Error(),
					})
				}
			}()
		}
		return nil
	}

	if err := s.aiService.GenerateStreamResponse(ctx, userInput, characterContext, conversationHistory, onChunk); err != nil {
		return WrapError(ErrCodeAIGenerationFailed, "生成流式AI响应失败", err)
	}

	if ttsBuffer.Len() > 0 {
		toSpeak := ttsBuffer.String()
		go func() {
			ttsCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			s.aiService.TextToSpeech(ttsCtx, toSpeak, sc)
		}()
	}

	sc.WriteJSON(map[string]any{
		"type":      "stream_end",
		"response":  fullResponse.String(),
		"timestamp": time.Now(),
	})
	return nil
}
