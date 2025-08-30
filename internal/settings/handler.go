package settings

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"q7o/internal/common/response"
	"q7o/internal/common/validator"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

// GetSettings получает настройки текущего пользователя
// @Summary Get user settings
// @Description Get current user's settings
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=SettingsResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/settings [get]
func (h *Handler) GetSettings(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	settings, err := h.service.GetUserSettings(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, settings)
}

// UpdateSettings обновляет настройки текущего пользователя
// @Summary Update user settings
// @Description Update current user's settings
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param settings body UpdateSettingsDTO true "Settings to update"
// @Success 200 {object} response.Response{data=SettingsResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/settings [put]
func (h *Handler) UpdateSettings(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	var dto UpdateSettingsDTO
	if err := c.BodyParser(&dto); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Validate request
	if err := validator.ValidateStruct(&dto); err != nil {
		return response.ValidationError(c, err)
	}

	settings, err := h.service.UpdateUserSettings(userID, &dto)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, settings)
}

// DeleteSettings удаляет настройки пользователя (сброс к значениям по умолчанию)
// @Summary Delete user settings
// @Description Delete current user's settings (reset to defaults)
// @Tags settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/settings [delete]
func (h *Handler) DeleteSettings(c *fiber.Ctx) error {
	userIDStr := c.Locals("userID").(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid user ID")
	}

	err = h.service.DeleteUserSettings(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	// После удаления создаём новые настройки по умолчанию
	settings, err := h.service.GetUserSettings(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, settings)
}