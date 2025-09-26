package infra

import (
	"context"

	"github.com/google/uuid"
	"github.com/justin/echome-be/gen/gen/model"
	"github.com/justin/echome-be/gen/gen/query"
	"github.com/justin/echome-be/internal/domain"
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
			character.Avatar = charModel.Avatar
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
		character.Avatar = charModel.Avatar
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
	if character.Avatar != nil {
		charModel.Avatar = character.Avatar
	}
	
	// 使用Create或Save方法保存
	err := r.query.Character.WithContext(ctx).Save(charModel)
	if err != nil {
		return err
	}
	
	return nil
}

var _ domain.CharacterRepository = (*CharacterRepository)(nil)

// Update 更新角色
func (r *CharacterRepository) Update(ctx context.Context, character *domain.Character) error {
	// 创建更新字段的map
	updateFields := make(map[string]any)
	
	// 添加需要更新的字段
	updateFields["name"] = character.Name
	updateFields["prompt"] = character.Prompt
	updateFields["flag"] = character.Flag
	updateFields["status"] = character.Status
	
	// 处理可空字段
	if character.Avatar != nil {
		updateFields["avatar"] = character.Avatar
	}
	
	// 如果有Description字段，也添加到更新map中
	if character.Description != nil {
		updateFields["description"] = character.Description
	}
		
	// 使用map更新字段
	_, err := r.query.Character.WithContext(ctx).
		Where(r.query.Character.ID.Eq(character.ID.String())).
		Updates(updateFields)
	if err != nil {
		return err
	}
	
	return nil
}

// NewCharacterRepository 创建新的CharacterRepository实例
func NewCharacterRepository() *CharacterRepository {
	return &CharacterRepository{
	query: GetQuery(),
	}
}

// GetCharactersByStatus 根据状态获取角色
func (r *CharacterRepository) GetCharactersByStatus(ctx context.Context, status int32) ([]*domain.Character, error) {
	charModels, err := r.query.Character.WithContext(ctx).Where(r.query.Character.Status.Eq(status)).Find()
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
			ID:        id,
			Name:      charModel.Name,
			Prompt:    charModel.Prompt,
			CreatedAt: charModel.CreatedAt,
			UpdatedAt: charModel.UpdatedAt,
			Flag:      charModel.Flag,
			Status:    charModel.Status,
		}
		
		// 处理可空字段
		if charModel.Avatar != nil {
			character.Avatar = charModel.Avatar
		}
		if charModel.Description != nil {
			character.Description = charModel.Description
		}
		if charModel.Voice != nil {
			character.Voice = *charModel.Voice
		}
		
		characters = append(characters, character)
	}
	
	return characters, nil
}
