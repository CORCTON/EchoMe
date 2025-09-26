package infra

import (
	"context"
	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/gen/gen/model"
	"github.com/justin/echome-be/gen/gen/query"
)

// CharacterRepository 实现domain.CharacterRepository接口

type CharacterRepository struct {
	query *query.Query
}

// GetAll 获取所有角色
func (r *CharacterRepository) GetAll(ctx context.Context) ([]*domain.Character, error) {
	charModels, err := r.query.Character.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	
	// 转换为domain.Character切片
	characters := make([]*domain.Character, 0, len(charModels))
	for _, charModel := range charModels {
		id, err := uuid.Parse(charModel.ID)
		if err != nil {
			return nil, err
		}
		
		character := &domain.Character{
			ID: id,
			Name: charModel.Name,
			Prompt: charModel.Prompt,
			CreatedAt: charModel.CreatedAt,
			UpdatedAt: charModel.UpdatedAt,
		}
		
		// 处理可空字段
		if charModel.Avatar != nil {
			character.Avatar = *charModel.Avatar
		}
		
		characters = append(characters, character)
	}
	
	return characters, nil
}

// GetByID 根据ID获取角色
func (r *CharacterRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Character, error) {
	charModel, err := r.query.Character.WithContext(ctx).Where(r.query.Character.ID.Eq(id.String())).First()
	if err != nil {
		return nil, err
	}
	
	// 转换为domain.Character
	character := &domain.Character{
		ID: id,
		Name: charModel.Name,
		Prompt: charModel.Prompt,
		CreatedAt: charModel.CreatedAt,
		UpdatedAt: charModel.UpdatedAt,
	}
	
	// 处理可空字段
	if charModel.Avatar != nil {
		character.Avatar = *charModel.Avatar
	}
	
	return character, nil
}

// Save 保存角色
func (r *CharacterRepository) Save(ctx context.Context, character *domain.Character) error {
	// 转换为model.Character
	charModel := &model.Character{
		ID: character.ID.String(),
		Name: character.Name,
		Prompt: character.Prompt,
		CreatedAt: character.CreatedAt,
		UpdatedAt: character.UpdatedAt,
	}
	
	// 处理可空字段
	if character.Avatar != "" {
		avatar := character.Avatar
		charModel.Avatar = &avatar
	}
	
	// 使用Create或Save方法保存
	err := r.query.Character.WithContext(ctx).Save(charModel)
	if err != nil {
		return err
	}
	
	return nil
}

var _ domain.CharacterRepository = (*CharacterRepository)(nil)

// NewCharacterRepository 创建新的CharacterRepository实例
func NewCharacterRepository() *CharacterRepository {
	return &CharacterRepository{
	query: GetQuery(),
	}
}
