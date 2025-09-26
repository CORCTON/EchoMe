package interfaces

import (
	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/internal/response"
	"github.com/labstack/echo/v4"
)

type CharacterHandlers struct {
	characterService domain.CharacterService
}

func NewCharacterHandlers(characterService domain.CharacterService) *CharacterHandlers {
	return &CharacterHandlers{
		characterService: characterService,
	}
}

// RegisterRoutes 注册角色相关路由
func (h *CharacterHandlers) RegisterRoutes(e *echo.Echo) {
	e.GET("/api/characters", h.GetCharacters)
	e.GET("/api/characters/:id", h.GetCharacterByID)
	e.POST("/api/character", h.CreateCharacter)
}

// GetCharacters handles GET /api/characters
// @Summary 获取所有角色
// @Description 获取所有可用角色的列表
// @Tags characters
// @Accept json
// @Produce json
// @Success 200 {array} domain.Character
// @Router /characters [get]
func (h *CharacterHandlers) GetCharacters(c echo.Context) error {
	characters, err := h.characterService.GetAllCharacters(c.Request().Context())
	if err != nil {
		return response.InternalError(c, "Failed to get characters", err.Error())
	}

	return response.Success(c, characters)
}


// GetCharacterByID handles GET /api/characters/:id
// @Summary 获取角色详情
// @Description 根据角色ID获取详细信息
// @Tags characters
// @Accept json
// @Produce json
// @Param id path string true "角色ID"
// @Success 200 {object} domain.Character
// @Router /characters/{id} [get]
func (h *CharacterHandlers) GetCharacterByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid character ID", err.Error())
	}

	character, err := h.characterService.GetCharacterByID(c.Request().Context(), id)
	if err != nil {
		return response.NotFound(c, "Character not found", err.Error())
	}

	return response.Success(c, character)
}


// CreateCharacter handles POST /api/characters/clone-voice
// @Summary 语音克隆并创建角色
// @Description 通过语音克隆创建带有克隆声音的角色
// @Tags characters
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "包含voiceCloneConfig和characterInfo的请求体"
// @Success 201 {object} domain.Character
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /characters/clone-voice [post]
func (h *CharacterHandlers) CreateCharacter(c echo.Context) error {
	var requestBody map[string]interface{}
	if err := c.Bind(&requestBody); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	// 解析voiceCloneConfig
	cloneConfigData, ok := requestBody["voiceCloneConfig"].(map[string]interface{})
	if !ok {
		return response.BadRequest(c, "Missing or invalid voiceCloneConfig", "voiceCloneConfig is required")
	}

	cloneConfig := domain.VoiceCloneConfig{
		AudioURL:         getStringField(cloneConfigData, "audioURL"),
		VoiceName:        getStringField(cloneConfigData, "voiceName"),
		VoiceDescription: getStringField(cloneConfigData, "voiceDescription"),
		LanguageType:     getStringField(cloneConfigData, "languageType"),
	}

	// 解析characterInfo
	characterInfoData, ok := requestBody["characterInfo"].(map[string]interface{})
	if !ok {
		return response.BadRequest(c, "Missing or invalid characterInfo", "characterInfo is required")
	}

	characterInfo := &domain.Character{
		Name:        getStringField(characterInfoData, "name"),
		Description: getStringField(characterInfoData, "description"),
		Persona:     getStringField(characterInfoData, "characterSetting"),
		AvatarURL:   getStringField(characterInfoData, "avatarURL"),
	}

	// 执行语音克隆并创建角色
	character, err := h.characterService.CreateCharacter(c.Request().Context(), &cloneConfig, characterInfo)
	if err != nil {
		return response.InternalError(c, "Failed to clone voice and create character", err.Error())
	}

	return response.Created(c, character)
}

// getStringField 从map中获取字符串字段，处理类型断言
func getStringField(data map[string]interface{}, field string) string {
	value, ok := data[field]
	if !ok {
		return ""
	}
	strValue, _ := value.(string)
	return strValue
}