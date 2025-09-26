package infra

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
	"gorm.io/gorm"
)

// CharacterModel 数据库模型
type CharacterModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(100);not null;unique"`
	Description string    `gorm:"type:text"`
	Persona     string    `gorm:"type:text;not null"`
	AvatarURL   string    `gorm:"type:text"`
	VoiceConfig string    `gorm:"type:jsonb"` // 存储JSON格式的VoiceProfile
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// CharacterRepository 实现domain.CharacterRepository接口
type CharacterRepository struct {
	db *gorm.DB
}

// 确保CharacterRepository实现了domain.CharacterRepository接口
var _ domain.CharacterRepository = (*CharacterRepository)(nil)

// NewCharacterRepository 创建一个新的CharacterRepository实例
func NewCharacterRepository(db *gorm.DB) *CharacterRepository {
	return &CharacterRepository{db: db}
}

// GetByID 根据ID获取角色
func (r *CharacterRepository) GetByID(id uuid.UUID) (*domain.Character, error) {
	var model CharacterModel
	result := r.db.Where("id = ?", id).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("character not found")
		}
		return nil, result.Error
	}

	// 转换为domain.Character
	character := &domain.Character{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		Persona:     model.Persona,
		AvatarURL:   model.AvatarURL,
	}

	// 解析VoiceConfig JSON
	if model.VoiceConfig != "" {
		var voiceProfile domain.VoiceProfile
		if err := json.Unmarshal([]byte(model.VoiceConfig), &voiceProfile); err != nil {
			return nil, fmt.Errorf("failed to unmarshal voice config: %w", err)
		}
		character.VoiceConfig = &voiceProfile
	}

	return character, nil
}

// GetAll 获取所有角色
func (r *CharacterRepository) GetAll() ([]*domain.Character, error) {
	var models []CharacterModel
	result := r.db.Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	// 转换为domain.Character列表
	characters := make([]*domain.Character, len(models))
	for i, model := range models {
		character := &domain.Character{
			ID:          model.ID,
			Name:        model.Name,
			Description: model.Description,
			Persona:     model.Persona,
			AvatarURL:   model.AvatarURL,
		}

		// 解析VoiceConfig JSON
		if model.VoiceConfig != "" {
			var voiceProfile domain.VoiceProfile
			if err := json.Unmarshal([]byte(model.VoiceConfig), &voiceProfile); err != nil {
				return nil, fmt.Errorf("failed to unmarshal voice config for character %s: %w", model.Name, err)
			}
			character.VoiceConfig = &voiceProfile
		}

		characters[i] = character
	}

	return characters, nil
}

// Save 保存角色
func (r *CharacterRepository) Save(character *domain.Character) error {
	// 检查角色ID是否为空，如果为空则生成一个新的
	if character.ID == uuid.Nil {
		character.ID = uuid.New()
	}

	// 转换为数据库模型
	model := CharacterModel{
		ID:          character.ID,
		Name:        character.Name,
		Description: character.Description,
		Persona:     character.Persona,
		AvatarURL:   character.AvatarURL,
	}

	// 将VoiceConfig序列化为JSON字符串
	if character.VoiceConfig != nil {
		voiceConfigJSON, err := json.Marshal(character.VoiceConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal voice config: %w", err)
		}
		model.VoiceConfig = string(voiceConfigJSON)
	}

	// 保存到数据库
	result := r.db.Save(&model)
	return result.Error
}
