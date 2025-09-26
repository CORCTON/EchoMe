package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Character struct {
	ID          uuid.UUID     `json:"id"`
	// 角色名
	Name        string        `json:"name"`
	// 角色提示词
	Description string        `json:"description"`
	// 角色性格描述
	Persona     string        `json:"persona"`
	// 角色头像URL
	AvatarURL   string        `json:"avatar_url"`
	// 角色声音配置
	VoiceConfig *VoiceProfile `json:"voice_config,omitempty" gorm:"type:jsonb"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// VoiceProfile 角色声音配置
type VoiceProfile struct {
	// Voice 对应克隆音色的voice_id
	Voice        string  `json:"voice"`
	// 语速 (0.5-2.0)      
	SpeechRate   float32 `json:"speech_rate"`    
	// 传入的时候需要字符串数组，但是只有第一个生效，简化为单字符串
	Language      string  `json:"language_hints"`    
}


type CharacterRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Character, error)
	GetAll(ctx context.Context) ([]*Character, error)
	Save(ctx context.Context, character *Character) error
}
type CharacterService interface {
	// GetCharacterByID 根据角色ID获取角色配置
	GetCharacterByID(ctx context.Context, id uuid.UUID) (*Character, error)
	GetAllCharacters(ctx context.Context) ([]*Character, error)
	// VoiceCloneAndCreateCharacter 执行语音克隆并创建带有克隆声音的角色
	// @param ctx 上下文
	// @param config 语音克隆配置
	// @param characterInfo 角色基本信息
	// @return 创建的角色
	CreateCharacter(ctx context.Context, config *VoiceCloneConfig, characterInfo *Character) (*Character, error)
}
