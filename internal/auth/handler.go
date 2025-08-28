package auth

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"q7o/internal/common/response"
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

func (h *Handler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, err)
	}

	user, tokens, err := h.service.Register(c.Context(), req)
	if err != nil {
		if err.Error() == "email already exists" || err.Error() == "username.go already exists" {
			return response.Conflict(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"user":          user,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, err)
	}

	user, tokens, err := h.service.Login(c.Context(), req)
	if err != nil {
		if err.Error() == "invalid credentials" {
			return response.Unauthorized(c, "Invalid email or password")
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"user":          user,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}

func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	tokens, err := h.service.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return response.Unauthorized(c, "Invalid refresh token")
	}

	return response.Success(c, tokens)
}

func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.VerifyEmail(c.Context(), req.Email, req.Code); err != nil {
		return response.BadRequest(c, "Invalid verification code")
	}

	return response.Success(c, fiber.Map{
		"message": "Email verified successfully",
	})
}

func (h *Handler) ResendVerification(c *fiber.Ctx) error {
	var req ResendVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.ResendVerification(c.Context(), req.Email); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Verification code sent",
	})
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req LogoutRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.Logout(c.Context(), userID, req.RefreshToken); err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Logged out successfully",
	})
}

func (h *Handler) ValidateToken(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	user, err := h.service.ValidateAndGetUser(c.Context(), userID)
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"valid": true,
		"user":  user,
	})
}

func (h *Handler) CheckUsername(c *fiber.Ctx) error {
	var req CheckUsernameRequest

	// Проверяем query параметры для GET запроса
	username := c.Query("username.go")
	if username != "" {
		req.Username = username
		req.FirstName = c.Query("first_name")
		req.LastName = c.Query("last_name")
	} else {
		// Иначе парсим body для POST запроса
		if err := c.BodyParser(&req); err != nil {
			return response.BadRequest(c, "Invalid request")
		}
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, err)
	}

	result, err := h.service.CheckUsernameAvailability(c.Context(), req.Username, req.FirstName, req.LastName)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, result)
}

func (h *Handler) SuggestUsernames(c *fiber.Ctx) error {
	var req SuggestUsernamesRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, err)
	}

	suggestions, err := h.service.GenerateUsernameSuggestions(c.Context(), req.FirstName, req.LastName)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"suggestions": suggestions,
	})
}

// Request DTOs
type RegisterRequest struct {
	FirstName string `json:"first_name" validate:"required,min=2,max=100"`
	LastName  string `json:"last_name" validate:"required,min=2,max=100"`
	Username  string `json:"username.go" validate:"omitempty,min=3,max=50"` // опциональный
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type CheckUsernameRequest struct {
	Username  string `json:"username.go" validate:"required,min=3,max=50"`
	FirstName string `json:"first_name" validate:"omitempty,min=2,max=100"`
	LastName  string `json:"last_name" validate:"omitempty,min=2,max=100"`
}

type SuggestUsernamesRequest struct {
	FirstName string `json:"first_name" validate:"required,min=2,max=100"`
	LastName  string `json:"last_name" validate:"required,min=2,max=100"`
}
