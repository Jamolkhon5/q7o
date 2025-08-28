package user

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"q7o/internal/common/response"

	"github.com/google/uuid"
)

type Handler struct {
	service  *Service
	validate *validator.Validate
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
	}
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	user, err := h.service.GetUserByID(c.Context(), uid)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, user)
}

func (h *Handler) GetUser(c *fiber.Ctx) error {
	id := c.Params("id")

	// Try parsing as UUID first
	if uid, err := uuid.Parse(id); err == nil {
		user, err := h.service.GetUserByID(c.Context(), uid)
		if err != nil {
			return response.BadRequest(c, "User not found")
		}
		return response.Success(c, user)
	}

	// Try as username.go
	user, err := h.service.GetUserByUsername(c.Context(), id)
	if err != nil {
		return response.BadRequest(c, "User not found")
	}

	return response.Success(c, user)
}

func (h *Handler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	var req UpdateUserDTO
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, err)
	}

	user, err := h.service.UpdateProfile(c.Context(), uid, &req)
	if err != nil {
		if err.Error() == "username already taken" {
			return response.Conflict(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, user)
}

func (h *Handler) SearchUsers(c *fiber.Ctx) error {
	query := c.Query("q", "")
	if query == "" {
		return response.BadRequest(c, "Search query required")
	}

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	users, err := h.service.SearchUsers(c.Context(), query, limit, offset)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"users": users,
		"count": len(users),
	})
}
