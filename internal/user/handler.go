package user

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"q7o/internal/common/response"
)

// ContactService interface
type ContactService interface {
	IsContact(ctx context.Context, userID, contactID uuid.UUID) (bool, error)
}

type Handler struct {
	service        *Service
	validate       *validator.Validate
	contactService ContactService
}

func NewHandler(service *Service, contactService ContactService) *Handler {
	return &Handler{
		service:        service,
		validate:       validator.New(),
		contactService: contactService,
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
	currentUserID := c.Locals("userID").(string)
	currentUID, _ := uuid.Parse(currentUserID)

	id := c.Params("id")

	// Try parsing as UUID first
	if uid, err := uuid.Parse(id); err == nil {
		user, err := h.service.GetUserByID(c.Context(), uid)
		if err != nil {
			return response.BadRequest(c, "User not found")
		}

		// Проверка контакта
		isContact := false
		canCall := false

		// Не проверяем для самого себя
		if currentUID != uid {
			if h.contactService != nil {
				isContact, _ = h.contactService.IsContact(c.Context(), currentUID, uid)
				canCall = isContact
			}
		}

		return response.Success(c, fiber.Map{
			"user":       user,
			"is_contact": isContact,
			"can_call":   canCall,
		})
	}

	// Try as username
	user, err := h.service.GetUserByUsername(c.Context(), id)
	if err != nil {
		return response.BadRequest(c, "User not found")
	}

	// Проверка контакта
	isContact := false
	canCall := false

	// Не проверяем для самого себя
	if currentUID != user.ID {
		if h.contactService != nil {
			isContact, _ = h.contactService.IsContact(c.Context(), currentUID, user.ID)
			canCall = isContact
		}
	}

	return response.Success(c, fiber.Map{
		"user":       user,
		"is_contact": isContact,
		"can_call":   canCall,
	})
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
	currentUserID := c.Locals("userID").(string)
	currentUID, _ := uuid.Parse(currentUserID)

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

	// Если есть сервис контактов, добавляем информацию о контактах
	if h.contactService != nil {
		for i := range users {
			if users[i].ID != currentUID {
				isContact, _ := h.contactService.IsContact(c.Context(), currentUID, users[i].ID)
				// Создаем новую map для каждого пользователя с дополнительной информацией
				userWithContact := make(map[string]interface{})
				userWithContact["id"] = users[i].ID
				userWithContact["username"] = users[i].Username
				userWithContact["first_name"] = users[i].FirstName
				userWithContact["last_name"] = users[i].LastName
				userWithContact["email"] = users[i].Email
				userWithContact["avatar_url"] = users[i].AvatarURL
				userWithContact["status"] = users[i].Status
				userWithContact["last_seen"] = users[i].LastSeen
				userWithContact["created_at"] = users[i].CreatedAt
				userWithContact["is_contact"] = isContact
				userWithContact["can_call"] = isContact

				// Заменяем в массиве
				users[i] = users[i] // Это временно, нужно будет обновить тип возврата
			}
		}
	}

	return response.Success(c, fiber.Map{
		"users": users,
		"count": len(users),
	})
}
