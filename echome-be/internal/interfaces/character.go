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


// CreateCharacter handles POST /api/character
// @Summary 创建角色（语音克隆）
// @Description 通过语音克隆创建角色
// @Tags characters
// @Accept json
// @Produce json
// @Param request body map[string]any true "包含audio（可选）、description(可选)、name（必须）、prompt（必须）、avatar（可选）和flag（必须）的请求体"
// @Success 201 {object} domain.Character
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /character [post]
func (h *CharacterHandlers) CreateCharacter(c echo.Context) error {
	var requestBody map[string]any
	if err := c.Bind(&requestBody); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	// 验证必填字段
	name, nameOk := requestBody["name"].(string)
	prompt, promptOk := requestBody["prompt"].(string)
	flag, flagOk := requestBody["flag"]
	if !nameOk || name == "" || !promptOk || prompt == "" || !flagOk {
		return response.BadRequest(c, "Missing required fields", "name, prompt and flag are required")
	}

	// 检查flag是否为布尔类型
	flagBool, ok := flag.(bool)
	if !ok {
		return response.BadRequest(c, "Invalid field type", "flag must be a boolean")
	}

	// 解析可选字段
	audio := getStringField(requestBody, "audio")

	avatar := getStringField(requestBody, "avatar")

	description := getStringField(requestBody, "description")

	// 创建角色信息
	characterInfo := &domain.Character{
		Name:        name,
		Prompt:      prompt,
		Avatar:      avatar,
		Description: description,
		Flag:        flagBool,
	}
		// 执行语音克隆并创建角色
		character, err := h.characterService.CreateCharacter(c.Request().Context(), audio, characterInfo)
		if err != nil {
			return response.InternalError(c, "Failed to clone voice and create character", err.Error())
		}

		return response.Created(c, character)
}

// getStringField 从map中获取字符串字段，处理类型断言
func getStringField(data map[string]any, field string) *string {
	value, ok := data[field]
	if !ok {
		return nil
	}
	strValue, _ := value.(string)
	return &strValue
}