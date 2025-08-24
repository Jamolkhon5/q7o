package meeting

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"q7o/internal/common/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateMeeting creates a new meeting room
func (h *Handler) CreateMeeting(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	var req CreateMeetingRequest
	if err := c.BodyParser(&req); err != nil {
		// If no body, create instant meeting with defaults
		req = CreateMeetingRequest{
			Title:        "Instant Meeting",
			AllowGuests:  true,
			RequiresAuth: false,
		}
	}

	meeting, err := h.service.CreateMeeting(c.Context(), uid, &req)
	if err != nil {
		return response.InternalError(c, err)
	}

	// Generate token for host
	token, err := h.service.livekit.GenerateMeetingToken(
		meeting.RoomName,
		userID,
		meeting.HostName,
		"host",
	)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"meeting": meeting,
		"token":   token,
		"ws_url":  h.service.cfg.PublicHost,
	})
}

// ValidateMeetingCode checks if a meeting code is valid
func (h *Handler) ValidateMeetingCode(c *fiber.Ctx) error {
	var req PreJoinCheckRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	result, err := h.service.ValidateMeetingCode(c.Context(), req.MeetingCode)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, result)
}

// JoinMeeting joins a meeting with a code
func (h *Handler) JoinMeeting(c *fiber.Ctx) error {
	// Check if user is authenticated
	var userID *uuid.UUID
	if uid, ok := c.Locals("userID").(string); ok && uid != "" {
		parsedID, _ := uuid.Parse(uid)
		userID = &parsedID
	}

	var req JoinMeetingRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	result, err := h.service.JoinMeeting(c.Context(), userID, &req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, result)
}

// JoinMeetingAuth joins a meeting as authenticated user
func (h *Handler) JoinMeetingAuth(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	var req JoinMeetingRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	result, err := h.service.JoinMeeting(c.Context(), &uid, &req)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, result)
}

// LeaveMeeting handles participant leaving
func (h *Handler) LeaveMeeting(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid meeting ID")
	}

	if err := h.service.LeaveMeeting(c.Context(), meetingID, uid); err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Left meeting successfully",
	})
}

// EndMeeting ends a meeting (host only)
func (h *Handler) EndMeeting(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid meeting ID")
	}

	if err := h.service.EndMeeting(c.Context(), meetingID, uid); err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, fiber.Map{
		"message": "Meeting ended",
	})
}

// GetMeetingParticipants returns current participants
func (h *Handler) GetMeetingParticipants(c *fiber.Ctx) error {
	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid meeting ID")
	}

	participants, err := h.service.GetMeetingParticipants(c.Context(), meetingID)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, participants)
}

// UpdateParticipantStatus updates participant's media status
func (h *Handler) UpdateParticipantStatus(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid meeting ID")
	}

	var req UpdateParticipantRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.service.UpdateParticipantStatus(c.Context(), meetingID, uid, &req); err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Status updated",
	})
}

// GetUserMeetings returns user's meeting history
func (h *Handler) GetUserMeetings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	meetings, err := h.service.GetUserMeetings(c.Context(), uid, limit, offset)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, meetings)
}
