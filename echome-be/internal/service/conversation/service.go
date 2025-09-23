package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
	"golang.org/x/sync/errgroup"
)

// ConversationService 会话服务实现
type ConversationService struct {
	aiService        domain.AIService
	characterService domain.CharacterService
	sessionService   domain.SessionService
	messageRepo      domain.MessageRepository
}

// NewConversationService 创建会话服务
func NewConversationService(
	aiService domain.AIService,
	characterService domain.CharacterService,
	sessionService domain.SessionService,
	messageRepo domain.MessageRepository,
) *ConversationService {
	return &ConversationService{
		aiService:        aiService,
		characterService: characterService,
		sessionService:   sessionService,
		messageRepo:      messageRepo,
	}
}

// StartVoiceConversation 开始语音会话
func (s *ConversationService) StartVoiceConversation(ctx context.Context, req *domain.VoiceConversationRequest) error {
	log.Printf("启动语音会话: session=%s character=%s", req.SessionID, req.CharacterID)

	if req.SessionID == uuid.Nil {
		return WrapError(ErrCodeInvalidInput, "缺少 SessionID", nil)
	}
	if req.CharacterID == uuid.Nil {
		return WrapError(ErrCodeInvalidInput, "缺少 CharacterID", nil)
	}
	if req.WebSocketConn == nil {
		return WrapError(ErrCodeInvalidInput, "缺少 WebSocket 连接", nil)
	}

	character, err := s.characterService.GetCharacterByID(req.CharacterID)
	if err != nil {
		return WrapError(ErrCodeCharacterNotFound, "角色不存在", err)
	}

	voiceConfig, err := s.GetCharacterVoiceConfig(req.CharacterID)
	if err != nil {
		return WrapError(ErrCodeConfigurationError, "获取角色语音配置失败", err)
	}

	session, err := s.sessionService.GetSessionByID(req.SessionID)
	if err != nil {
		return WrapError(ErrCodeSessionNotFound, "会话不存在", err)
	}

	log.Printf("语音会话已启动: session=%s character=%s", session.ID, character.Name)
	return s.handleVoiceConversationFlow(ctx, req.WebSocketConn, voiceConfig, session)
}

// 语音会话处理流程
func (s *ConversationService) handleVoiceConversationFlow(
	ctx context.Context,
	clientWS *websocket.Conn,
	voiceConfig *domain.VoiceConfig,
	session *domain.Session,
) error {
	log.Printf("处理语音会话: session=%s", session.ID)

	g, ctx := errgroup.WithContext(ctx)

	// 读消息循环
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				messageType, data, err := clientWS.ReadMessage()
				if err != nil {
					return err // 出错直接返回
				}

				switch messageType {
				case websocket.BinaryMessage:
					// 内联处理二进制语音消息
					log.Printf("收到语音消息: session=%s", session.ID)
					response := map[string]any{
						"type":       "audio_received",
						"message":    "语音已接收，开始处理",
						"session_id": session.ID,
					}
					responseData, err := json.Marshal(response)
					if err != nil {
						return fmt.Errorf("序列化响应失败: %w", err)
					}
					if err := clientWS.WriteMessage(websocket.TextMessage, responseData); err != nil {
						return err
					}
				case websocket.TextMessage:
					var textMsg map[string]any
					if err := json.Unmarshal(data, &textMsg); err != nil {
						continue
					}
					if text, ok := textMsg["text"].(string); ok {
						if err := s.processTextInput(ctx, text, voiceConfig, session, clientWS); err != nil {
							return err
						}
					}
				case websocket.CloseMessage:
					log.Printf("客户端关闭连接: session=%s", session.ID)
					return nil
				}
			}
		}
	})
	return g.Wait()
}

// 处理文本输入（AI -> TTS 流程）
func (s *ConversationService) processTextInput(ctx context.Context, text string, voiceConfig *domain.VoiceConfig, session *domain.Session, clientWS *websocket.Conn) error {
	log.Printf("收到文本输入: session=%s text=%s", session.ID, text)

	userMessage := &domain.Message{
		ID:        uuid.New(),
		SessionID: session.ID,
		Content:   text,
		Sender:    "user",
		Timestamp: time.Now(),
	}
	_ = s.messageRepo.Save(userMessage)

	sessionHistory, _ := s.messageRepo.GetBySessionID(session.ID)

	aiRequest := &domain.AIRequest{
		UserInput:        text,
		CharacterContext: voiceConfig.Character,
		SessionHistory:   sessionHistory,
		Language:         voiceConfig.Language,
	}

	aiResponse, err := s.generateAIResponse(ctx, aiRequest)
	if err != nil {
		// 内联错误响应处理
		errorResponse := map[string]interface{}{
			"type":       "error",
			"message":    "生成 AI 回复失败",
			"session_id": session.ID,
			"timestamp":  time.Now(),
		}
		responseData, _ := json.Marshal(errorResponse)
		return clientWS.WriteMessage(websocket.TextMessage, responseData)
	}

	aiMessage := &domain.Message{
		ID:        uuid.New(),
		SessionID: session.ID,
		Content:   aiResponse.Text,
		Sender:    "ai",
		Timestamp: time.Now(),
	}
	_ = s.messageRepo.Save(aiMessage)

	textResponse := map[string]interface{}{
		"type":       "text_response",
		"text":       aiResponse.Text,
		"message_id": aiMessage.ID,
		"session_id": session.ID,
		"timestamp":  aiMessage.Timestamp,
	}

	responseData, _ := json.Marshal(textResponse)
	return clientWS.WriteMessage(websocket.TextMessage, responseData)
}

// 调用 AI 服务生成回复
func (s *ConversationService) generateAIResponse(ctx context.Context, req *domain.AIRequest) (*domain.AIResponse, error) {
	characterContext := ""
	if req.CharacterContext != nil {
		characterContext = fmt.Sprintf("你是%s。%s。%s",
			req.CharacterContext.Name,
			req.CharacterContext.Description,
			req.CharacterContext.Persona)
	}

	responseText, err := s.aiService.GenerateResponse(ctx, req.UserInput, characterContext)
	if err != nil {
		return nil, fmt.Errorf("AI 服务错误: %w", err)
	}

	return &domain.AIResponse{
		Text: responseText,
		Metadata: map[string]interface{}{
			"character_id": req.CharacterContext.ID,
			"language":     req.Language,
			"timestamp":    time.Now(),
		},
	}, nil
}

// 文本消息处理（非实时接口）
func (s *ConversationService) ProcessTextMessage(ctx context.Context, req *domain.TextMessageRequest) (*domain.TextMessageResponse, error) {
	if req.SessionID == uuid.Nil {
		return nil, WrapError(ErrCodeInvalidInput, "缺少 SessionID", nil)
	}
	if req.CharacterID == uuid.Nil {
		return nil, WrapError(ErrCodeInvalidInput, "缺少 CharacterID", nil)
	}
	if req.UserInput == "" {
		return nil, WrapError(ErrCodeInvalidInput, "缺少用户输入", nil)
	}

	character, err := s.characterService.GetCharacterByID(req.CharacterID)
	if err != nil {
		return nil, WrapError(ErrCodeCharacterNotFound, "角色不存在", err)
	}

	sessionHistory, _ := s.messageRepo.GetBySessionID(req.SessionID)

	aiRequest := &domain.AIRequest{
		UserInput:        req.UserInput,
		CharacterContext: character,
		SessionHistory:   sessionHistory,
		Language:         "zh-CN",
	}

	aiResponse, err := s.generateAIResponse(ctx, aiRequest)
	if err != nil {
		return nil, WrapError(ErrCodeAIGenerationFailed, "生成 AI 回复失败", err)
	}

	userMessage := &domain.Message{
		ID:        uuid.New(),
		SessionID: req.SessionID,
		Content:   req.UserInput,
		Sender:    "user",
		Timestamp: time.Now(),
	}
	_ = s.messageRepo.Save(userMessage)

	aiMessage := &domain.Message{
		ID:        uuid.New(),
		SessionID: req.SessionID,
		Content:   aiResponse.Text,
		Sender:    "ai",
		Timestamp: time.Now(),
	}
	_ = s.messageRepo.Save(aiMessage)

	return &domain.TextMessageResponse{
		Response:  aiResponse.Text,
		MessageID: aiMessage.ID,
		Timestamp: aiMessage.Timestamp,
	}, nil
}

// 获取角色的语音配置
func (s *ConversationService) GetCharacterVoiceConfig(characterID uuid.UUID) (*domain.VoiceConfig, error) {
	character, err := s.characterService.GetCharacterByID(characterID)
	if err != nil {
		return nil, fmt.Errorf("获取角色失败: %w", err)
	}

	voiceConfig := &domain.VoiceConfig{
		Character: character,
		ASRConfig: domain.ASRConfig{
			Model:      "paraformer-realtime-v2",
			Format:     "pcm",
			SampleRate: 16000,
		},
		TTSConfig: domain.TTSConfig{
			Model:          "qwen-tts-realtime",
			Voice:          "Cherry",
			ResponseFormat: "pcm",
			SampleRate:     24000,
			Mode:           "server_commit",
		},
		Language: "zh-CN",
	}

	if character.VoiceConfig != nil {
		voiceConfig.TTSConfig.Voice = character.VoiceConfig.Voice
		voiceConfig.TTSConfig.SampleRate = int(character.VoiceConfig.SpeechRate * 24000)
		voiceConfig.Language = character.VoiceConfig.Language
		log.Printf("角色 %s 使用自定义语音参数: %v", character.Name, character.VoiceConfig.CustomParams)
	}

	return voiceConfig, nil
}
