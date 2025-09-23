package infrastructure

import (
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
)

type MemoryCharacterRepository struct {
	characters map[uuid.UUID]*domain.Character
	nameIndex  map[string]uuid.UUID
	mu         sync.RWMutex
}

func NewMemoryCharacterRepository() *MemoryCharacterRepository {
	repo := &MemoryCharacterRepository{
		characters: make(map[uuid.UUID]*domain.Character),
		nameIndex:  make(map[string]uuid.UUID),
	}

	// 添加一些预设角色用于测试
	repo.initializeDefaultCharacters()

	return repo
}

// initializeDefaultCharacters 初始化默认角色
func (r *MemoryCharacterRepository) initializeDefaultCharacters() {
	defaultCharacters := []*domain.Character{
		{
			ID:          uuid.New(),
			Name:        "小助手",
			Description: "友善的AI助手",
			Persona:     "你是一个友善、耐心的AI助手，总是乐于帮助用户解决问题。你说话温和，回答详细且有用。",
			AvatarURL:   "",
			VoiceConfig: &domain.VoiceProfile{
				Voice:         "xiaoyun",  // 阿里云小云语音
				SpeakingStyle: "friendly", // 友善风格
				SpeechRate:    1.0,        // 正常语速
				Pitch:         0,          // 正常音调
				Volume:        0.8,        // 80%音量
				Language:      "zh-CN",    // 中文
				CustomParams: map[string]string{
					"emotion": "gentle",
				},
			},
		},
		{
			ID:          uuid.New(),
			Name:        "专业顾问",
			Description: "专业的技术顾问",
			Persona:     "你是一个专业的技术顾问，具有丰富的技术知识和经验。你的回答准确、专业，善于用简单的语言解释复杂的技术概念。",
			AvatarURL:   "",
			VoiceConfig: &domain.VoiceProfile{
				Voice:         "zhiwei",       // 阿里云志伟语音（男声）
				SpeakingStyle: "professional", // 专业风格
				SpeechRate:    0.9,            // 稍慢语速
				Pitch:         -50,            // 稍低音调
				Volume:        0.9,            // 90%音量
				Language:      "zh-CN",        // 中文
				CustomParams: map[string]string{
					"emotion":   "calm",
					"formality": "formal",
				},
			},
		},
		{
			ID:          uuid.New(),
			Name:        "创意伙伴",
			Description: "富有创意的思维伙伴",
			Persona:     "你是一个富有创意和想象力的伙伴，善于从不同角度思考问题，提供创新的解决方案和有趣的想法。",
			AvatarURL:   "",
			VoiceConfig: &domain.VoiceProfile{
				Voice:         "xiaogang", // 阿里云小刚语音（活泼）
				SpeakingStyle: "cheerful", // 活泼风格
				SpeechRate:    1.1,        // 稍快语速
				Pitch:         50,         // 稍高音调
				Volume:        0.85,       // 85%音量
				Language:      "zh-CN",    // 中文
				CustomParams: map[string]string{
					"emotion":    "excited",
					"creativity": "high",
				},
			},
		},
		{
			ID:          uuid.New(),
			Name:        "English Tutor",
			Description: "Professional English language tutor",
			Persona:     "You are a professional English tutor with native-level fluency. You speak clearly and help users improve their English skills with patience and encouragement.",
			AvatarURL:   "",
			VoiceConfig: &domain.VoiceProfile{
				Voice:         "jenny",       // 阿里云Jenny英文语音
				SpeakingStyle: "educational", // 教育风格
				SpeechRate:    0.85,          // 较慢语速便于学习
				Pitch:         0,             // 正常音调
				Volume:        0.9,           // 90%音量
				Language:      "en-US",       // 美式英语
				CustomParams: map[string]string{
					"accent":  "american",
					"clarity": "high",
				},
			},
		},
		{
			ID:          uuid.New(),
			Name:        "故事讲述者",
			Description: "富有表现力的故事讲述者",
			Persona:     "你是一个富有表现力的故事讲述者，善于用生动的语言和丰富的情感讲述各种故事。你的声音充满魅力，能够吸引听众的注意力。",
			AvatarURL:   "",
			VoiceConfig: &domain.VoiceProfile{
				Voice:         "xiaomeng",  // 阿里云小梦语音（温柔女声）
				SpeakingStyle: "narrative", // 叙述风格
				SpeechRate:    0.95,        // 稍慢语速
				Pitch:         25,          // 稍高音调
				Volume:        0.9,         // 90%音量
				Language:      "zh-CN",     // 中文
				CustomParams: map[string]string{
					"emotion":      "expressive",
					"storytelling": "dramatic",
				},
			},
		},
	}

	for _, character := range defaultCharacters {
		r.characters[character.ID] = character
		r.nameIndex[character.Name] = character.ID
	}
}

// GetByID 根据ID获取角色
func (r *MemoryCharacterRepository) GetByID(id uuid.UUID) (*domain.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	character, exists := r.characters[id]
	if !exists {
		return nil, errors.New("character not found")
	}

	return character, nil
}

// GetByName 根据名称获取角色
func (r *MemoryCharacterRepository) GetByName(name string) (*domain.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.nameIndex[name]
	if !exists {
		return nil, errors.New("character not found")
	}

	character, exists := r.characters[id]
	if !exists {
		return nil, errors.New("character not found")
	}

	return character, nil
}

// GetAll 获取所有角色
func (r *MemoryCharacterRepository) GetAll() ([]*domain.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var characters []*domain.Character
	for _, character := range r.characters {
		characters = append(characters, character)
	}

	return characters, nil
}

// Search 根据查询字符串搜索角色
func (r *MemoryCharacterRepository) Search(query string) ([]*domain.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if query == "" {
		return r.getAllCharacters(), nil
	}

	var results []*domain.Character
	queryLower := strings.ToLower(query)

	// 模糊搜索：支持名称、描述、角色设定的部分匹配
	for _, character := range r.characters {
		if r.matchesQuery(character, queryLower) {
			results = append(results, character)
		}
	}

	return results, nil
}

// matchesQuery 检查角色是否匹配查询条件
func (r *MemoryCharacterRepository) matchesQuery(character *domain.Character, queryLower string) bool {
	return strings.Contains(strings.ToLower(character.Name), queryLower) ||
		strings.Contains(strings.ToLower(character.Description), queryLower) ||
		strings.Contains(strings.ToLower(character.Persona), queryLower)
}

// getAllCharacters 获取所有角色（内部方法，不加锁）
func (r *MemoryCharacterRepository) getAllCharacters() []*domain.Character {
	var characters []*domain.Character
	for _, character := range r.characters {
		characters = append(characters, character)
	}
	return characters
}

// Save 新建角色
func (r *MemoryCharacterRepository) Save(character *domain.Character) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.characters[character.ID] = character
	r.nameIndex[character.Name] = character.ID

	return nil
}
