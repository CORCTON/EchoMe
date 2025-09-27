package conversation

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/client/aliyun"
	"github.com/justin/echome-be/internal/domain"
)

// WebSocketRequest 定义 WebSocket 请求结构体
type WebSocketRequest struct {
	Messages []domain.ContextMessage `json:"messages"`
	Stream   bool                    `json:"stream,omitempty"`
}

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
		character, err = s.characterService.GetCharacterByID(ctx, req.CharacterID)
		if err != nil {
			zap.L().Warn("获取角色失败", zap.Error(err), zap.String("characterID", req.CharacterID.String()))
			// 获取角色失败不应中断流程，可以继续无角色的对话
			character = nil
		}
	}
	return s.handleVoiceConversationFlow(ctx, req.SafeConn, character)
}

// handleVoiceConversationFlow 处理语音对话流程
func (s *ConversationService) handleVoiceConversationFlow(ctx context.Context, sc domain.WebSocketConn, character *domain.Character) error {
	// 使用 WithCancel 创建可以被 errgroup 控制的上下文
	g, ctx := errgroup.WithContext(ctx)
	defer func() {
		if err := g.Wait(); err != nil {
			zap.L().Error("errgroup 等待时出错", zap.Error(err))
		}
	}()

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// 继续执行
			}

			messageType, message, err := sc.ReadMessage()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					zap.L().Info("读取超时，客户端无响应，关闭连接")
					return nil // 正常关闭
				}
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					zap.L().Info("客户端关闭连接", zap.Error(err))
					return nil // 正常关闭
				}
				zap.L().Warn("读取WebSocket消息失败", zap.Error(err))
				return err // 返回错误以停止 errgroup
			}

			if messageType != websocket.TextMessage {
				zap.L().Warn("预期文本消息，收到类型不匹配", zap.Int("message_type", messageType))
				continue
			}

			var req WebSocketRequest
			if err := json.Unmarshal(message, &req); err != nil {
				zap.L().Warn("解析JSON消息失败", zap.Error(err), zap.String("raw_message", string(message)))
				_ = sc.WriteJSON(map[string]any{
					"type":    "error",
					"message": "无效的请求格式，需要JSON",
				})
				continue
			}

			if len(req.Messages) == 0 {
				zap.L().Warn("请求消息为空")
				_ = sc.WriteJSON(map[string]any{
					"type":    "error",
					"message": "请求消息不能为空",
				})
				continue
			}

			//  对话上下文
			cctx := make([]map[string]string, 0, len(req.Messages))
			for _, msg := range req.Messages {
				cctx = append(cctx, map[string]string{
					"role":    msg.Role,
					"content": msg.Content,
				})
			}
			zap.L().Debug("解析到用户输入和对话历史", zap.Int("history_count", len(cctx)))

			// 根据 stream 字段决定处理流程
			if req.Stream {
				if err := s.handleStreamingConversation(ctx, sc, cctx, character); err != nil {
					zap.L().Error("流式对话处理失败", zap.Error(err))
					continue
				}
			} else {
				response, err := s.aiService.GenerateResponse(ctx, cctx)
				if err != nil {
					zap.L().Error("生成AI响应失败", zap.Error(err))
					_ = sc.WriteJSON(map[string]any{
						"type":    "error",
						"message": "生成AI响应失败",
					})
					continue
				}

				_ = sc.WriteJSON(map[string]any{
					"type":      "text_response",
					"response":  response,
					"timestamp": time.Now(),
				})
			}
		}
	})

	return g.Wait()
}

// handleStreamingConversation 处理流式对话
func (s *ConversationService) handleStreamingConversation(
	ctx context.Context,
	sc domain.WebSocketConn,
	cctx []map[string]string,
	character *domain.Character, // 传入整个 character 对象以获取语音信息
) error {
	_ = sc.WriteJSON(map[string]any{"type": "stream_start", "timestamp": time.Now()})

	llmTextChan := make(chan string, 100) // Buffered channel for LLM text chunks

	ttsConfig := aliyun.DefaultTTSConfig()
	if character != nil && character.Flag && character.Voice != nil {
		ttsConfig.Voice = *character.Voice
	}

	g, ctx := errgroup.WithContext(ctx)

	// Goroutine 1: 处理TTS流
	g.Go(func() error {
		return s.aiService.HandleCosyVoiceTTS(ctx, sc, llmTextChan, ttsConfig)
	})

	// Goroutine 2: 生成LLM响应并发送到channel
	g.Go(func() error {
		defer close(llmTextChan)

		onChunk := func(chunk string) error {
			if chunk == "" {
				return nil
			}

			// 将文本块发送给客户端用于显示
			if err := sc.WriteJSON(map[string]any{
				"type":      "stream_chunk",
				"content":   chunk,
				"timestamp": time.Now(),
			}); err != nil {
				zap.L().Warn("向WebSocket写入流式块失败", zap.Error(err))
				return err // 返回错误，停止errgroup
			}

			// 将文本块发送到TTS channel
			select {
			case llmTextChan <- chunk:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}

		return s.aiService.GenerateStreamResponse(ctx, cctx, onChunk)
	})

	if err := g.Wait(); err != nil {
		if err != context.Canceled && err != context.DeadlineExceeded {
			zap.L().Error("流式处理 errgroup 遇到错误", zap.Error(err))
			_ = sc.WriteJSON(map[string]any{
				"type":    "error",
				"message": "流式响应处理失败: " + err.Error(),
			})
			return err
		}
	}

	_ = sc.WriteJSON(map[string]any{
		"type":      "stream_end",
		"timestamp": time.Now(),
	})
	return nil
}