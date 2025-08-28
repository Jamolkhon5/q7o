package contact

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

func (h *Handler) SendContactRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	var dto SendContactRequestDTO
	if err := c.BodyParser(&dto); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&dto); err != nil {
		return response.ValidationError(c, err)
	}

	request, err := h.service.SendContactRequest(c.Context(), uid, &dto)
	if err != nil {
		if err.Error() == "already in contacts" || err.Error() == "request already exists" {
			return response.Conflict(c, err.Error())
		}
		if err.Error() == "user not found" {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, request)
}

func (h *Handler) AcceptContactRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	requestID := c.Params("request_id")
	reqID, err := uuid.Parse(requestID)
	if err != nil {
		return response.BadRequest(c, "Invalid request ID")
	}

	if err := h.service.AcceptContactRequest(c.Context(), uid, reqID); err != nil {
		if err.Error() == "unauthorized" {
			return response.Unauthorized(c, err.Error())
		}
		if err.Error() == "request not found" {
			return response.BadRequest(c, err.Error())
		}
		if err.Error() == "request already processed" {
			return response.Conflict(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Contact request accepted",
	})
}

func (h *Handler) RejectContactRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	requestID := c.Params("request_id")
	reqID, err := uuid.Parse(requestID)
	if err != nil {
		return response.BadRequest(c, "Invalid request ID")
	}

	if err := h.service.RejectContactRequest(c.Context(), uid, reqID); err != nil {
		if err.Error() == "unauthorized" {
			return response.Unauthorized(c, err.Error())
		}
		if err.Error() == "request not found" {
			return response.BadRequest(c, err.Error())
		}
		if err.Error() == "request already processed" {
			return response.Conflict(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Contact request rejected",
	})
}

func (h *Handler) RemoveContact(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	contactID := c.Params("contact_id")
	cid, err := uuid.Parse(contactID)
	if err != nil {
		return response.BadRequest(c, "Invalid contact ID")
	}

	if err := h.service.RemoveContact(c.Context(), uid, cid); err != nil {
		if err.Error() == "not in contacts" {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Contact removed",
	})
}

func (h *Handler) GetContacts(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	contacts, err := h.service.GetContacts(c.Context(), uid, limit, offset)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"contacts": contacts,
		"count":    len(contacts),
	})
}

func (h *Handler) GetContactRequests(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	requestType := c.Query("type", "incoming") // incoming or outgoing
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	var requests []*ContactRequestWithUser
	var err error

	if requestType == "outgoing" {
		requests, err = h.service.GetOutgoingRequests(c.Context(), uid, limit, offset)
	} else {
		requests, err = h.service.GetIncomingRequests(c.Context(), uid, limit, offset)
	}

	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"requests": requests,
		"count":    len(requests),
		"type":     requestType,
	})
}

func (h *Handler) CheckContact(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	checkUserID := c.Params("user_id")
	checkUID, err := uuid.Parse(checkUserID)
	if err != nil {
		return response.BadRequest(c, "Invalid user ID")
	}

	isContact, err := h.service.IsContact(c.Context(), uid, checkUID)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"is_contact": isContact,
		"user_id":    checkUserID,
	})
}
