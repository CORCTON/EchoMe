package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/client/aliyun"
	"github.com/justin/echome-be/internal/domain"
)

// ttsTask 表示一个TTS任务
type ttsTask struct {
	text     string
	ttsCtx   context.Context
	cancel   context.CancelFunc
	sc       domain.WebSocketConn
	taskID   int
	taskType string
	// 角色声音配置
	voiceProfile *domain.VoiceProfile
}

// ConversationService 会话服务实现
type ConversationService struct {
	aiService        domain.AIService
	characterService domain.CharacterService

	// 为每个WebSocket连接维护一个TTS任务队列
	ttsTaskQueues map[string]chan ttsTask
	queueMutex    sync.Mutex
}

// NewConversationService 创建会话服务
func NewConversationService(
	aiService domain.AIService,
	characterService domain.CharacterService,
) *ConversationService {
	return &ConversationService{
		aiService:        aiService,
		characterService: characterService,
		ttsTaskQueues:    make(map[string]chan ttsTask),
	}
}
var _ domain.ConversationService = (*ConversationService)(nil)

// getOrCreateTTSQueue 为给定的连接获取或创建TTS任务队列
func (s *ConversationService) getOrCreateTTSQueue(connectionID string, ctx context.Context) chan ttsTask {
	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()

	// 检查是否已存在队列
	queue, exists := s.ttsTaskQueues[connectionID]
	if !exists {
		// 创建新队列
		queue = make(chan ttsTask, 100)
		s.ttsTaskQueues[connectionID] = queue

		// 启动处理协程
		go s.processTTSTasks(ctx, connectionID, queue)
	}

	return queue
}

// processTTSTasks 按顺序处理TTS任务
func (s *ConversationService) processTTSTasks(ctx context.Context, connectionID string, queue chan ttsTask) {
	var lastTaskID int = -1

	for {
		select {
		case <-ctx.Done():
			// 上下文取消，清理队列
			s.queueMutex.Lock()
			delete(s.ttsTaskQueues, connectionID)
			s.queueMutex.Unlock()
			close(queue)
			return
		case task, ok := <-queue:
			if !ok {
				// 队列已关闭
				return
			}

			// 确保任务按顺序执行
			if task.taskID > lastTaskID+1 {
				log.Printf("Warning: Skipping TTS task with ID %d (expected %d)", task.taskID, lastTaskID+1)
				continue
			}

			lastTaskID = task.taskID

			// 如果角色声音配置不为空，使用克隆声音专用配置
			if task.voiceProfile != nil {
				// 使用克隆声音专用配置
				cfg := domain.TTSConfig{
					Voice: task.voiceProfile.Voice,
				}
				if err := s.aiService.TextToSpeech(task.ttsCtx, task.text, task.sc, cfg); err != nil {
					_ = task.sc.WriteJSON(map[string]any{
						"type":    "tts_error",
						"message": err.Error(),
					})
					log.Printf("TTS error for task %s: %v", task.taskType, err)
				}
			} else {
				// 使用普通的TTS
				if err := s.aiService.TextToSpeech(task.ttsCtx, task.text, task.sc, aliyun.DefaultTTSConfig()); err != nil {
					_ = task.sc.WriteJSON(map[string]any{
						"type":    "tts_error",
						"message": err.Error(),
					})
					log.Printf("TTS error for task %s: %v", task.taskType, err)
				}
			}
			defer task.cancel()
		}
	}
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

	// 如果提供了角色ID，先尝试获取角色
	if req.CharacterID != uuid.Nil {
		character, err = s.characterService.GetCharacterByID(req.CharacterID)
		if err != nil {
			log.Printf("Character not found with ID %v, will not use character: %v", req.CharacterID, err)
			character = nil
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

	// 如果有角色，使用角色描述作为上下文；否则使用空字符串
	characterContext := ""
	if character != nil {
		characterContext = character.Description
	}

	// 使用AI服务生成响应，传入前端提供的对话历史
	response, err := s.aiService.GenerateResponse(ctx, req.UserInput, characterContext, conversationHistory)
	if err != nil {
		return nil, WrapError(ErrCodeAIGenerationFailed, "生成AI响应失败", err)
	}

	// 创建响应对象
	return &domain.TextMessageResponse{
			Response:  response,
			MessageID: uuid.New(),
			Timestamp: time.Now(),
		},
		nil
}

// StartVoiceConversation 开始语音会话
func (s *ConversationService) StartVoiceConversation(ctx context.Context, req *domain.VoiceConversationRequest) error {
	var character *domain.Character
	var err error

	// 如果提供了角色ID，先尝试获取角色
	if req.CharacterID != uuid.Nil {
		character, err = s.characterService.GetCharacterByID(req.CharacterID)
		if err != nil {
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
			log.Printf("Error reading WebSocket message: %v", err)

			// 检查是否是"超时"错误，这表示客户端已失联
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
			Text     string                  `json:"text"`
			Messages []domain.ContextMessage `json:"messages,omitempty"`
			Stream   bool                    `json:"stream,omitempty"` // 是否启用流式响应
		}

		userInput := string(message)
		// 初始化对话历史
		conversationHistory := []map[string]string{}

		// 尝试解析JSON结构
		if err := json.Unmarshal(message, &structuredMessage); err == nil {
			// 成功解析为结构化消息
			if structuredMessage.Text != "" {
				userInput = structuredMessage.Text
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
			// 非结构化消息，跳过处理
			continue
		}

		// 如果有角色，传入角色描述作为上下文
		characterContext := ""
		if character != nil {
			characterContext = character.Description
		}

		// 根据是否启用流式响应选择不同的处理方式
		if structuredMessage.Stream {
			if err := s.handleStreamingConversation(conversationCtx, sc, userInput, character, conversationHistory); err != nil {
				_ = sc.WriteJSON(map[string]any{
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
	// 直接用 sc 写，保证不会并发写 conn
	_ = sc.WriteJSON(map[string]any{"type": "stream_start", "timestamp": time.Now()})

	var fullResponse, ttsBuffer strings.Builder
	const ttsMinLen = 50

	// 为这个连接生成唯一ID并获取任务队列
	connectionID := fmt.Sprintf("stream_%d", time.Now().UnixNano())
	queue := s.getOrCreateTTSQueue(connectionID, ctx)
	var taskID = 0

	onChunk := func(chunk string) error {
		fullResponse.WriteString(chunk)
		_ = sc.WriteJSON(map[string]any{
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

			// 创建TTS上下文
			ttsCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

			// 将TTS任务添加到队列
			task := ttsTask{
				text:        toSpeak,
				ttsCtx:      ttsCtx,
				cancel:      cancel,
				sc:          sc,
				taskID:      taskID,
				taskType:    "stream_chunk",
			}
			// 只有当character不为nil时才设置voiceProfile
			if character != nil {
				task.voiceProfile = character.VoiceConfig
			}
			queue <- task
			taskID++
		}
		return nil
	}

	// 如果有角色，使用角色描述作为上下文；否则使用空字符串
	characterContext := ""
	if character != nil {
		characterContext = character.Description
	}

	if err := s.aiService.GenerateStreamResponse(ctx, userInput, characterContext, conversationHistory, onChunk); err != nil {
		return WrapError(ErrCodeAIGenerationFailed, "生成流式AI响应失败", err)
	}

	if ttsBuffer.Len() > 0 {
		toSpeak := ttsBuffer.String()

		// 创建TTS上下文
		ttsCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

		// 将最后一个TTS任务添加到队列
		task := ttsTask{
			text:        toSpeak,
			ttsCtx:      ttsCtx,
			cancel:      cancel,
			sc:          sc,
			taskID:      taskID,
			taskType:    "stream_final",
		}
		// 只有当character不为nil时才设置voiceProfile
		if character != nil {
			task.voiceProfile = character.VoiceConfig
		}
		queue <- task
		taskID++
	}

	_ = sc.WriteJSON(map[string]any{
		"type":      "stream_end",
		"response":  fullResponse.String(),
		"timestamp": time.Now(),
	})
	return nil
}
