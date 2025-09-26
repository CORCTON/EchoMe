package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VoiceProfile struct {
	Voice string `json:"voice"`
}

type Character struct {
	ID          uuid.UUID     `json:"id"`
	// 角色名
	Name        string        `json:"name"`
	// 角色提示词
	Prompt string        `json:"prompt"`
	// 角色头像URL
	Avatar   string        `json:"avatar"`
	AvatarURL string        `json:"avatar_url"`
	// 角色描述
	Description string        `json:"description"`
	// 角色设定
	Persona     string        `json:"character_setting"`
	// 角色声音配置
	VoiceConfig *VoiceProfile `json:"voice_config,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
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
