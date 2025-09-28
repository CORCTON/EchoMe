package character

import (
	"context"

	"github.com/google/uuid"
)

// Repo 角色仓库接口
type Repo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Character, error)
	GetAll(ctx context.Context) ([]*Character, error)
	Save(ctx context.Context, character *Character) error
	Update(ctx context.Context, character *Character) error
	// GetCharactersByStatus 根据状态获取角色列表
	GetCharactersByStatus(ctx context.Context, status int32) ([]*Character, error)
}
