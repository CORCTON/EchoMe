package conversation

import (
	"context"
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
		Language:  "zh", // 默认语言，可从角色配置或参数中获取
	}, nil
}

// ProcessTextMessage 处理文本消息并返回AI响应
func (s *ConversationService) ProcessTextMessage(ctx context.Context, req *domain.TextMessageRequest) (*domain.TextMessageResponse, error) {
	// 在实际实现中，这里应该获取用户的当前角色或默认角色
	// 为了简化演示，我们假设使用默认角色ID
	defaultCharacterID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")

	// 获取角色信息
	character, err := s.characterService.GetCharacterByID(defaultCharacterID)
	if err != nil {
		return nil, WrapError(ErrCodeCharacterNotFound, "获取角色失败", err)
	}

	// 使用AI服务生成响应，传入空的对话历史
	response, err := s.aiService.GenerateResponse(ctx, req.UserInput, character.Description, []map[string]string{})
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
	if req.CharacterID == uuid.Nil {
		return WrapError(ErrCodeInvalidInput, "缺少角色ID", nil)
	}
	if req.WebSocketConn == nil {
		return WrapError(ErrCodeInvalidInput, "缺少 WebSocket 连接", nil)
	}

	// 获取角色
	character, err := s.characterService.GetCharacterByID(req.CharacterID)
	if err != nil {
		// 如果角色不存在，创建一个默认角色
		log.Printf("Character not found, using default character: %v", err)
		character = &domain.Character{
			ID:          req.CharacterID,
			Name:        "默认角色",
			Description: "你是一个友好的AI助手，乐于回答用户的问题。",
		}
	}
	return s.handleSimpleVoiceConversationFlow(ctx, req.WebSocketConn, character, req.Language)
}

// handleSimpleVoiceConversationFlow 处理简单的语音对话流程
func (s *ConversationService) handleSimpleVoiceConversationFlow(ctx context.Context, conn *websocket.Conn, character *domain.Character, language string) error {
	log.Println("Starting voice conversation flow with character:", character.ID)

	// 为对话创建一个新的上下文
	conversationCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer conn.Close()

	// 初始化对话历史
	conversationHistory := []map[string]string{}

	// 设置对话历史长度限制，防止历史记录过长
	const maxHistoryLength = 10 // 保留5轮对话（10条消息：5个用户，5个助手）

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

		// 处理文本输入
		userInput := string(message)
		log.Printf("Processing user input: %s", userInput)

		// 使用AI服务生成响应，传入对话历史
		characterContext := character.Description
		log.Println("Generating AI response based on character context and conversation history")
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

		// 更新对话历史
		// 添加当前用户消息
		conversationHistory = append(conversationHistory, map[string]string{
			"role":    "user",
			"content": userInput,
		})
		// 添加AI响应消息
		conversationHistory = append(conversationHistory, map[string]string{
			"role":    "assistant",
			"content": response,
		})

		// 如果对话历史超过限制，移除最早的消息
		if len(conversationHistory) > maxHistoryLength {
			// 保留最新的消息，移除最早的两条（一条用户，一条助手）
			conversationHistory = conversationHistory[2:]
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
