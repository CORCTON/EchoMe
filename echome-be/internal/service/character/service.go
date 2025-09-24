package character

import (
	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
)

// characterService implements domain.CharacterService
type characterService struct {
	characterRepo domain.CharacterRepository
}

// NewCharacterService creates a new character service
func NewCharacterService(repo domain.CharacterRepository) domain.CharacterService {
	return &characterService{
		characterRepo: repo,
	}
}

// GetCharacterByID retrieves a character by ID
func (s *characterService) GetCharacterByID(id uuid.UUID) (*domain.Character, error) {
	return s.characterRepo.GetByID(id)
}

// GetAllCharacters retrieves all characters
func (s *characterService) GetAllCharacters() ([]*domain.Character, error) {
	return s.characterRepo.GetAll()
}

// CreateCharacter creates a new character
func (s *characterService) CreateCharacter(character *domain.Character) error {
	if character.ID == uuid.Nil {
		character.ID = uuid.New()
	}
	return s.characterRepo.Save(character)
}
