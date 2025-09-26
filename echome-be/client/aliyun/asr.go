package aliyun

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/justin/echome-be/internal/domain"
	"golang.org/x/sync/errgroup"
)

// DefaultASRConfig 返回默认的阿里云WebSocket实时ASR配置
func DefaultASRConfig() domain.ASRConfig {
	return domain.ASRConfig{
		Model:         "paraformer-realtime-v2",
		Format:        "pcm",
		SampleRate:    16000,
		LanguageHints: []string{"zh", "en"},
	}
}

// sendHeartbeat 定期发送心跳消息保持连接活跃
func sendHeartbeat(ctx context.Context, ws *websocket.Conn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 发送ping消息
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(1*time.Second)); err != nil {
				return
			}
		}
	}
}

// HandleASR 通过阿里云Model Studio Paraformer处理语音识别
func (client *AliClient) HandleASR(ctx context.Context, clientWS domain.WebSocketConn) error {
	// 连接到阿里云Model Studio ASR WebSocket
	asrWS, taskID, err := connectToModelStudioASR(client.apiKey, DefaultASRConfig())
	if err != nil {
		return fmt.Errorf("连接Model Studio ASR失败: %w", err)
	}
	defer asrWS.Close()

	g, ctx := errgroup.WithContext(ctx)

	// 启动心跳goroutine，每15秒发送一次ping
	g.Go(func() error {
		sendHeartbeat(ctx, asrWS, 15*time.Second)
		return nil
	})

	// 从客户端读取音频并发送到阿里云
	g.Go(func() error {
		return forwardAudioToModelStudio(ctx, clientWS, asrWS, taskID)
	})

	// 从阿里云读取识别结果
	g.Go(func() error {
		return handleModelStudioASRResults(ctx, asrWS, clientWS)
	})

	return g.Wait()
}

// connectToModelStudioASR 连接到阿里云WebSocket实时ASR服务
func connectToModelStudioASR(apiKey string, config domain.ASRConfig) (*websocket.Conn, string, error) {
	// 阿里云WebSocket实时ASR URL
	url := "wss://dashscope.aliyuncs.com/api-ws/v1/inference"

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // 临时禁用验证
		},
	}

	headers := http.Header{}
	headers.Add("Authorization", "bearer "+apiKey)
	headers.Add("user-agent", "EchoMe/1.0")
	headers.Add("X-DashScope-DataInspection", "enable")
	ws, _, err := dialer.Dial(url, headers)
	if err != nil {
		return nil, "", fmt.Errorf("连接WebSocket ASR失败: %v", err)
	}

	zap.L().Info("WebSocket ASR连接成功")

	// 生成UUID格式的任务ID
	taskID := uuid.New().String()

	// 根据文档发送初始化参数 - 启用心跳机制
	initMsg := map[string]any{
		"header": map[string]any{
			"action":    "run-task",
			"task_id":   taskID,
			"streaming": "duplex",
		},
		"payload": map[string]any{
			"task_group": "audio",
			"task":       "asr",
			"function":   "recognition",
			"model":      config.Model,
			"parameters": map[string]any{
				"format":         config.Format,
				"sample_rate":    config.SampleRate,
				"language_hints": config.LanguageHints,
				"heartbeat":      true,
			},
			"input": map[string]any{},
		},
	}

	// 设置写入超时
	if err := ws.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		zap.L().Warn("设置写入超时失败", zap.Error(err))
	}

	if err := ws.WriteJSON(initMsg); err != nil {
		return nil, "", fmt.Errorf("发送初始化消息失败: %w", err)
	}

	// 设置读取超时
	if err := ws.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		zap.L().Warn("设置读取超时失败", zap.Error(err))
	}

	// 等待初始化响应
	_, initResp, err := ws.ReadMessage()
	if err != nil {
		return nil, "", fmt.Errorf("读取初始化响应失败: %w", err)
	}

	zap.L().Debug("ASR初始化响应", zap.String("response", string(initResp)))

	// 检查初始化是否成功
	var initResponse map[string]any
	if err := json.Unmarshal(initResp, &initResponse); err == nil {
		if header, ok := initResponse["header"].(map[string]any); ok {
			if event, ok := header["event"].(string); ok && event == "task-started" {
				zap.L().Info("ASR任务初始化成功")
				// 保存task_id在websocket的上下文，以便后续使用
				ws.SetPongHandler(func(string) error { return nil })
			} else if event == "task-failed" {
				errorCode := "未知错误"
				errorMessage := "任务初始化失败"
				if code, ok := header["error_code"].(string); ok {
					errorCode = code
				}
				if msg, ok := header["error_message"].(string); ok {
					errorMessage = msg
				}
				return nil, "", fmt.Errorf("ASR初始化失败: %s - %s", errorCode, errorMessage)
			}
		}
	}

	return ws, taskID, nil
}

// forwardAudioToModelStudio 转发音频数据到阿里云WebSocket ASR
func forwardAudioToModelStudio(ctx context.Context, clientWS domain.WebSocketConn, asrWS *websocket.Conn, taskID string) error {
	defer func() {
		zap.L().Info("音频发送结束，发送finish-task指令")
		// 发送结束信号 - 根据文档格式，使用相同的taskID
		endMsg := map[string]any{
			"header": map[string]any{
				"action":    "finish-task",
				"task_id":   taskID,
				"streaming": "duplex",
			},
			"payload": map[string]any{
				"input": map[string]any{},
			},
		}

		// 设置写入超时
		if err := asrWS.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			zap.L().Warn("设置写入超时失败", zap.Error(err))
		}

		if err := asrWS.WriteJSON(endMsg); err != nil {
			zap.L().Warn("发送finish-task指令失败", zap.Error(err))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("上下文已取消，停止音频转发")
			return ctx.Err()
		default:
			messageType, data, err := clientWS.ReadMessage()
			if err != nil {
				zap.L().Warn("从客户端读取消息失败", zap.Error(err))
				return nil
			}

			switch messageType {
			case websocket.BinaryMessage:
				// 设置写入超时
				if err := asrWS.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
					zap.L().Warn("设置写入超时失败", zap.Error(err))
				}

				if err := asrWS.WriteMessage(websocket.BinaryMessage, data); err != nil {
					return fmt.Errorf("转发音频数据失败: %w", err)
				}
			case websocket.TextMessage:
				var msg map[string]any
				if err := json.Unmarshal(data, &msg); err == nil {
					if msgType, ok := msg["type"].(string); ok && msgType == "finish" {
						zap.L().Info("收到客户端结束信号")
						return nil
					}
				} else {
					zap.L().Warn("无法解析文本消息", zap.Error(err), zap.String("message", string(data)))
				}
			}
		}
	}
}

// handleModelStudioASRResults 处理阿里云WebSocket ASR识别结果
func handleModelStudioASRResults(ctx context.Context, asrWS *websocket.Conn, clientWS domain.WebSocketConn) error {
	resultReceived := false

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("上下文已取消，停止处理ASR结果")
			return ctx.Err()
		default:
			// 设置读取超时
			if err := asrWS.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
				zap.L().Warn("设置读取超时失败", zap.Error(err))
			}

			msgType, msg, err := asrWS.ReadMessage()
			if err != nil {
				// 根据WebSocket错误类型进行处理
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					zap.L().Info("ASR连接正常关闭", zap.Error(err))
				} else {
					zap.L().Error("ASR连接异常断开", zap.Error(err))
				}
				return nil
			}

			// 确保只处理文本消息
			if msgType != websocket.TextMessage {
				continue
			}

			var response map[string]any
			if err := json.Unmarshal(msg, &response); err != nil {
				zap.L().Warn("解析ASR响应失败", zap.Error(err), zap.String("raw_message", string(msg)))
				continue
			}

			// 根据文档解析响应格式
			if header, ok := response["header"].(map[string]any); ok {
				if event, ok := header["event"].(string); ok {
					switch event {
					case "result-generated":
						// 处理识别结果
						resultReceived = true
						if payload, ok := response["payload"].(map[string]any); ok {
							if output, ok := payload["output"].(map[string]any); ok {
								if sentence, ok := output["sentence"].(map[string]any); ok {
									if text, ok := sentence["text"].(string); ok && text != "" {
										// 检查是否是heartbeat消息，如果是则跳过
										if heartbeat, ok := sentence["heartbeat"].(bool); ok && heartbeat {
											continue
										}
										// 构建客户端响应
										clientResponse := map[string]any{
											"type": "asr_result",
											"text": text,
										}

										// 添加句子结束标志
										if sentenceEnd, ok := sentence["sentence_end"].(bool); ok {
											clientResponse["sentence_end"] = sentenceEnd
										}

										// 设置写入超时
										if err := clientWS.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
											zap.L().Warn("设置客户端写入超时失败", zap.Error(err))
										}

										// 发送到客户端
										if err := clientWS.WriteJSON(clientResponse); err != nil {
											zap.L().Warn("向客户端发送ASR结果失败", zap.Error(err))
										}
									}
								}
							}
						}
					case "task-finished":
						zap.L().Info("ASR任务完成")
						if !resultReceived {
							zap.L().Warn("任务完成但未收到任何识别结果")
						}
						// 发送任务完成通知给客户端
						_ = clientWS.WriteJSON(map[string]any{
							"type": "asr_finished",
						})
						return nil
					case "task-failed":
						// 根据文档，错误信息在header中
						errorCode := "未知错误"
						errorMessage := "任务执行失败"
						if code, ok := header["error_code"].(string); ok {
							errorCode = code
						}
						if msg, ok := header["error_message"].(string); ok {
							errorMessage = msg
						}
						zap.L().Error("ASR任务失败", zap.String("error_code", errorCode), zap.String("error_message", errorMessage))
						// 发送错误信息给客户端
						_ = clientWS.WriteJSON(map[string]any{
							"type":          "asr_error",
							"error_code":    errorCode,
							"error_message": errorMessage,
						})
						return fmt.Errorf("ASR任务失败: %s - %s", errorCode, errorMessage)
					default:
					}
				}
			}
		}
	}
}
