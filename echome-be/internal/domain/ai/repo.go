package ai

import (
	"context"

	"github.com/justin/echome-be/internal/domain/ws"
)

type Repo interface {
	GetVoiceStatus(ctx context.Context, voiceID string) (bool, error)
	VoiceClone(ctx context.Context, url string) (*string, error)
	HandleCosyVoiceTTS(ctx context.Context, clientWS ws.WebSocketConn, textStream <-chan string, config TTSConfig) error
	GenerateResponse(ctx context.Context, msg DashScopeChatRequest, onChunk func(string) error) error
	PerformSearch(ctx context.Context, query string, apiKey string) (string, error)
	HandleASR(ctx context.Context, clientWS ws.WebSocketConn) error
}
