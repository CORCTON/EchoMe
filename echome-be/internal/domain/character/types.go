package character

import (
	"time"

	"github.com/google/uuid"
)

// 角色状态枚举定义
const (
	CharacterStatusPending  = 1 // 审核中
	CharacterStatusApproved = 2 // 可用
	CharacterStatusDisabled = 3 // 禁用
)

// Character 角色实体
type Character struct {
	ID uuid.UUID `json:"id"`
	// 角色名
	Name string `json:"name"`
	// 角色描述
	Description *string `json:"description"`
	// 角色提示词
	Prompt string `json:"prompt"`
	// 角色头像URL
	Avatar *string `json:"avatar"`
	// 角色音色
	Voice *string `json:"voice"`
	// 是否克隆音色
	Flag bool `json:"flag"`
	// AudioExample 音色示例音频URL
	AudioExample *string `json:"audio_example"`
	// 音色状态，使用枚举值: 1-审核中, 2-可用, 3-禁用
	Status    int32     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
