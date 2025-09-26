package character

import (
	"context"

	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
)

// CharacterService implements domain.CharacterService
type CharacterService struct {
	characterRepo domain.CharacterRepository
	aiService    domain.AIService
}

// NewCharacterService creates a new character service
func NewCharacterService(repo domain.CharacterRepository, aiService domain.AIService) *CharacterService {
	return &CharacterService{
		characterRepo: repo,
		aiService:    aiService,
	}
}

// GetCharacterByID 获取角色信息
func (s *CharacterService) GetCharacterByID(id uuid.UUID) (*domain.Character, error) {
	return s.characterRepo.GetByID(id)
}

// GetAllCharacters 获取所有角色
func (s *CharacterService) GetAllCharacters() ([]*domain.Character, error) {
	return s.characterRepo.GetAll()
}

// CreateCharacter 实现语音克隆并创建角色
func (s *CharacterService) CreateCharacter(ctx context.Context, config *domain.VoiceCloneConfig, characterInfo *domain.Character) (*domain.Character, error) {
	// 1. 执行语音克隆
	voiceID, err := s.aiService.VoiceClone(ctx, config)
	if err != nil {
		return nil, err
	}
	
	// 2. 设置角色的声音配置
	if characterInfo.VoiceConfig == nil {
		characterInfo.VoiceConfig = &domain.VoiceProfile{}
	}
	characterInfo.VoiceConfig.Voice = *voiceID
	
	// 3. 如果没有指定ID，则生成新ID
	if characterInfo.ID == uuid.Nil {
		characterInfo.ID = uuid.New()
	}
	
	// 4. 保存角色
	if err := s.characterRepo.Save(characterInfo); err != nil {
		return nil, err
	}
	
	return characterInfo, nil
}
