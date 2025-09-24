package conversation

import (
	"context"
	"encoding/json"
	"log"
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
	if req.WebSocketConn == nil {
		return WrapError(ErrCodeInvalidInput, "缺少 WebSocket 连接", nil)
	}

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
		// 将在handleSimpleVoiceConversationFlow中从消息获取实际角色ID
		character = &domain.Character{
			ID:          uuid.New(),
			Name:        "临时默认角色",
			Description: "你是一个友好的AI助手，乐于回答用户的问题。",
		}
	}

	return s.handleSimpleVoiceConversationFlow(ctx, req.WebSocketConn, character, req.Language)
}

// handleSimpleVoiceConversationFlow 处理语音对话流程
func (s *ConversationService) handleSimpleVoiceConversationFlow(ctx context.Context, conn *websocket.Conn, character *domain.Character, language string) error {
	log.Println("Starting voice conversation flow with character:", character.ID)

	// 为对话创建一个新的上下文
	conversationCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer conn.Close()

	// 循环处理WebSocket消息
	for {
		// 读取客户端发送的文本消息
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			// 客户端关闭连接是正常的
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return nil
			}
			return WrapError(ErrCodeWebSocketError, "读取WebSocket消息失败", err)
		}

		// 确保接收到的是文本消息
		if messageType != websocket.TextMessage {
			log.Printf("Expected text message but received %d", messageType)
			continue
		}

		// 处理输入消息
		var structuredMessage struct {
			Text         string             `json:"text"`
			CharacterID  string             `json:"character_id,omitempty"`
			Messages     []domain.ContextMessage `json:"messages,omitempty"`
		}
		
		userInput := string(message)
		conversationHistory := []map[string]string{}
		
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
		} else {
			// 非结构化消息，直接作为用户输入
			log.Printf("Received plain text message")
		}
		
		log.Printf("Processing user input: %s", userInput)

		// 使用AI服务生成响应，传入前端提供的对话上下文
		characterContext := character.Description
		log.Println("Generating AI response based on character context")
		response, err := s.aiService.GenerateResponse(conversationCtx, userInput, characterContext, conversationHistory)
		if err != nil {
			log.Printf("Failed to generate AI response: %v", err)
			return WrapError(ErrCodeAIGenerationFailed, "生成AI响应失败", err)
		}

		log.Printf("AI response generated: %s", response)

		// 先发送文本响应确认给客户端
		if err := conn.WriteJSON(map[string]interface{}{
			"type":      "text_response",
			"response":  response,
			"timestamp": time.Now(),
		}); err != nil {
			log.Printf("Failed to send text response: %v", err)
			return WrapError(ErrCodeWebSocketError, "发送文本响应失败", err)
		}
		// 在单独的goroutine中处理TTS，避免阻塞主消息循环
		go func(text string) {
			// 创建新的上下文，避免主上下文取消影响TTS处理
			ttsCtx, ttsCancel := context.WithCancel(context.Background())
			defer ttsCancel()

			log.Printf("Handling TTS to convert response to speech in background: %s", text)
			// 使用新的TextToSpeech方法直接处理文本到语音转换
			if err := s.aiService.TextToSpeech(ttsCtx, text, conn); err != nil {
				log.Printf("TTS handling error in background: %v", err)
				// 只记录错误，不中断主流程
				return
			}

			log.Println("Background TTS processing completed successfully")
		}(response)

		log.Println("Voice conversation round completed successfully")
	}
}
