package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VoiceProfile struct {
	Voice string `json:"voice"`
}

// 角色状态枚举定义
const (
	CharacterStatusPending = 1 // 审核中
	CharacterStatusApproved = 2 // 可用
	CharacterStatusDisabled = 3 // 禁用
)

// Character represents a role in the system
type Character struct {
	ID          uuid.UUID     `json:"id"`
	// 角色名
	Name        string        `json:"name"`
	// 角色描述
	Description *string        `json:"description"`
	// 角色提示词
	Prompt string        `json:"prompt"`
	// 角色头像URL
	Avatar   *string        `json:"avatar"`
	// 角色音色
	Voice   string        `json:"voice"`
	// 是否克隆音色
	Flag bool          `json:"flag"`
	// 音色状态，使用枚举值: 1-审核中, 2-可用, 3-禁用
	Status int32        `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}


// CharacterRepository 角色仓库接口
type CharacterRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Character, error)
	GetAll(ctx context.Context) ([]*Character, error)
	Save(ctx context.Context, character *Character) error
	Update(ctx context.Context, character *Character) error
	// GetCharactersByStatus 根据状态获取角色列表
	GetCharactersByStatus(ctx context.Context, status int32) ([]*Character, error)
}

type CharacterService interface {
	GetCharacterByID(ctx context.Context, id uuid.UUID) (*Character, error)
	GetAllCharacters(ctx context.Context) ([]*Character, error)
	CreateCharacter(ctx context.Context, audio *string, characterInfo *Character) (*Character, error)
	// CheckAndUpdatePendingCharacters 检查并更新审核中角色的状态
	CheckAndUpdatePendingCharacters(ctx context.Context) error
	// UpdateCharacterStatus 更新角色状态
	UpdateCharacterStatus(ctx context.Context, character *Character, status int32) error
}
