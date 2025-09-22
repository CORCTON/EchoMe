package interfaces

import (
	"github.com/google/uuid"
	"github.com/justin/echome-be/internal/domain"
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
	e.GET("/api/characters/search", h.SearchCharacters)
	e.GET("/api/characters/:id", h.GetCharacterByID)
	e.POST("/api/characters", h.CreateCharacter)
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
	characters, err := h.characterService.GetCharacterByID(uuid.Nil)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, characters)
}

// SearchCharacters handles GET /api/characters/search
// @Summary Search roles
// @Description Search for roles by query string
// @Tags characters
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Success 200 {array} domain.Character
// @Failure 500 {object} map[string]string
// @Router /characters/search [get]
func (h *CharacterHandlers) SearchCharacters(c echo.Context) error {
	query := c.QueryParam("q")
	characters, err := h.characterService.SearchCharacters(query)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, characters)
}

// GetCharacterByID handles GET /api/characters/:id
// @Summary Get character by ID
// @Description Get detailed information about a specific character
// @Tags characters
// @Accept json
// @Produce json
// @Param id path string true "Character ID"
// @Success 200 {object} domain.Character
// @Router /characters/{id} [get]
func (h *CharacterHandlers) GetCharacterByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid character ID"})
	}

	character, err := h.characterService.GetCharacterByID(id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Character not found"})
	}

	return c.JSON(200, character)
}

// CreateCharacter handles POST /api/characters
func (h *CharacterHandlers) CreateCharacter(c echo.Context) error {
	var character domain.Character
	if err := c.Bind(&character); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	if err := h.characterService.CreateCharacter(&character); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, character)
}
