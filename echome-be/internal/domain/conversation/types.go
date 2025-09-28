package conversation

import (
	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain/ai"
	"github.com/justin/echome-be/internal/domain/character"
	"github.com/justin/echome-be/internal/domain/ws"
)

// VoiceConversationRequest 语音对话请求
type VoiceConversationRequest struct {
	SafeConn    ws.WebSocketConn `json:"-"`
	CharacterID uuid.UUID        `json:"character_id"`
}

// VoiceConfig 角色的语音配置
type VoiceConfig struct {
	Character *character.Character `json:"character"`
	ASRConfig ai.ASRConfig         `json:"asr_config"`
	TTSConfig ai.TTSConfig         `json:"tts_config"`
	Language  string               `json:"language"`
}
