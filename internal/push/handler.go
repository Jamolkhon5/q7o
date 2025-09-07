package push

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"q7o/internal/common/response"
	"q7o/internal/common/validator"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterToken регистрирует токен устройства для push уведомлений
// POST /api/v1/push/register
func (h *Handler) RegisterToken(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	var req RegisterTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Используем валидатор напрямую
	if err := validator.ValidateStruct(req); err != nil {
		return response.ValidationError(c, err)
	}

	if err := h.service.RegisterDeviceToken(c.Context(), uid, &req); err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Device token registered successfully",
	})
}

// DeactivateToken деактивирует токен устройства
// POST /api/v1/push/deactivate
func (h *Handler) DeactivateToken(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	var req struct {
		Token string `json:"token" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Используем валидатор напрямую
	if err := validator.ValidateStruct(req); err != nil {
		return response.ValidationError(c, err)
	}

	if err := h.service.DeactivateToken(c.Context(), uid, req.Token); err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Device token deactivated successfully",
	})
}
