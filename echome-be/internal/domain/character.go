package domain

import "github.com/google/uuid"

type Character struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Persona     string        `json:"persona"`
	AvatarURL   string        `json:"avatar_url"`
	VoiceConfig *VoiceProfile `json:"voice_config,omitempty"`
}

// VoiceProfile defines the voice configuration for a character
type VoiceProfile struct {
	Voice         string            `json:"voice"`          // Aliyun TTS voice ID
	SpeakingStyle string            `json:"speaking_style"` // Speaking style
	SpeechRate    float32           `json:"speech_rate"`    // Speech rate (0.5-2.0)
	Pitch         float32           `json:"pitch"`          // Pitch adjustment (-500 to 500)
	Volume        float32           `json:"volume"`         // Volume (0.0-1.0)
	Language      string            `json:"language"`       // Language code (zh-CN, en-US, etc.)
	CustomParams  map[string]string `json:"custom_params"`  // Custom parameters for TTS
}
type CharacterRepository interface {
	GetByID(id uuid.UUID) (*Character, error)
	GetByName(name string) (*Character, error)
	GetAll() ([]*Character, error)
	Search(query string) ([]*Character, error)
	Save(character *Character) error
}
type CharacterService interface {
	GetCharacterByID(id uuid.UUID) (*Character, error)
	GetAllCharacters() ([]*Character, error)
	SearchCharacters(query string) ([]*Character, error)
	CreateCharacter(character *Character) error
}
