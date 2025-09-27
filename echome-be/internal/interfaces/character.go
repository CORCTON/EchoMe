package interfaces

import (
	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
	"github.com/justin/echome-be/internal/response"
	"github.com/labstack/echo/v4"
)

// CreateCharacterRequest 定义创建角色请求体结构
type CreateCharacterRequest struct {
	AudioExample *string `json:"audio_example"` // 可选，音频示例
	Audio       *string `json:"audio"`       // 可选，音频文件
	Description *string `json:"description"` // 可选，角色描述
	Name        string `json:"name"`        // 必须，角色名称
	Prompt      string `json:"prompt"`      // 必须，角色提示词
	Avatar      *string `json:"avatar"`      // 可选，角色头像
	Flag        bool   `json:"flag"`        // 必须，标志位
}

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
// @Router /api/characters [get]
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
// @Router /api/characters/{id} [get]
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
// @Param request body interfaces.CreateCharacterRequest true "创建角色的请求体参数"
// @Success 200
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/character [post]
func (h *CharacterHandlers) CreateCharacter(c echo.Context) error {
	var requestBody CreateCharacterRequest
	if err := c.Bind(&requestBody); err != nil {
		return response.BadRequest(c, "Invalid request body", err.Error())
	}

	// 验证必填字段
	if requestBody.Name == "" || requestBody.Prompt == "" {
		return response.BadRequest(c, "Missing required fields", "name, prompt and flag are required")
	}

	// 创建角色信息
	characterInfo := &domain.Character{
		Name:        requestBody.Name,
		Prompt:      requestBody.Prompt,
		Avatar:      requestBody.Avatar,
		Description: requestBody.Description,
		AudioExample: requestBody.Audio,
		Flag:        requestBody.Flag,
	}
	// 执行语音克隆并创建角色
	err := h.characterService.CreateCharacter(c.Request().Context(), requestBody.Audio, characterInfo)
	if err != nil {
		return response.InternalError(c, "Failed to clone voice and create character", err.Error())
	}

	return response.Success(c, uuid.Nil)
}