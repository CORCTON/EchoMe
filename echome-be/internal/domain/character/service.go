package character

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain/ai"
	"github.com/samber/lo"
)

// CharacterService 角色服务
type CharacterService struct {
	characterRepo Repo
	aiClient      ai.Repo
}

// NewCharacterService 创建角色服务
func NewCharacterService(repo Repo, aiClient ai.Repo) *CharacterService {
	return &CharacterService{
		characterRepo: repo,
		aiClient:      aiClient,
	}
}

// GetCharacterByID 获取角色信息
func (s *CharacterService) GetCharacterByID(ctx context.Context, id uuid.UUID) (*Character, error) {
	return s.characterRepo.GetByID(ctx, id)
}

// GetAllCharacters 获取所有角色
func (s *CharacterService) GetAllCharacters(ctx context.Context) ([]*Character, error) {
	return s.characterRepo.GetAll(ctx)
}

// CreateCharacter 创建角色
func (s *CharacterService) CreateCharacter(ctx context.Context, audio *string, characterInfo *Character) error {
	// 1. 角色初始化
	character := &Character{
		Name:         characterInfo.Name,
		Description:  characterInfo.Description,
		Prompt:       characterInfo.Prompt,
		Avatar:       characterInfo.Avatar,
		Flag:         characterInfo.Flag,
		AudioExample: characterInfo.AudioExample,
		Status:       CharacterStatusPending, // 使用枚举值设置初始状态为审核中
	}

	// 2. 判断是否需要创建音色
	if characterInfo.Flag {
		//  调用AI服务创建音色
		voiceProfile, err := s.aiClient.VoiceClone(ctx, lo.FromPtr(audio))
		if err != nil {
			return err
		}
		character.Voice = voiceProfile
	}
	err := s.characterRepo.Save(ctx, character)
	if err != nil {
		return err
	}
	return nil
}

// UpdateCharacterStatus 更新角色状态
func (s *CharacterService) UpdateCharacterStatus(ctx context.Context, character *Character, status int32) error {
	character.Status = status
	character.UpdatedAt = time.Now()
	return s.characterRepo.Update(ctx, character)
}

// CheckAndUpdatePendingCharacters 检查并更新审核中角色的状态
func (s *CharacterService) CheckAndUpdatePendingCharacters(ctx context.Context) error {
	// 获取所有审核中的角色
	pendingCharacters, err := s.characterRepo.GetCharactersByStatus(ctx, CharacterStatusPending)
	if err != nil {
		return err
	}

	// 遍历角色，检查音色状态
	for _, character := range pendingCharacters {
		if character.Flag && character.Voice != nil {
			// 查询音色状态
			status, err := s.aiClient.GetVoiceStatus(ctx, *character.Voice)
			if err != nil {
				// 如果查询失败，继续处理下一个角色
				continue
			}

			// 更新角色状态
			if status {
				// 音色审核通过
				_ = s.UpdateCharacterStatus(ctx, character, CharacterStatusApproved)
			}
		}
	}

	return nil
}
