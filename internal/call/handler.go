package call

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"log"
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

// GetCallToken генерирует токен для подключения к комнате LiveKit
func (h *Handler) GetCallToken(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	username := c.Locals("username").(string)

	var req TokenRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	uid, _ := uuid.Parse(userID)

	// Просто генерируем токен для подключения к комнате
	token, err := h.service.GenerateCallToken(req.RoomName, uid, username)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"token":     token,
		"room_name": req.RoomName,
		"ws_url":    h.service.GetLiveKitURL(),
		"user": fiber.Map{
			"id":       userID,
			"username": username,
		},
	})
}

// InitiateCall создает новый звонок
func (h *Handler) InitiateCall(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	username := c.Locals("username").(string)

	var req InitiateCallRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	callerID, _ := uuid.Parse(userID)
	calleeID, _ := uuid.Parse(req.CalleeID)

	// Создаем запись о звонке
	call, err := h.service.InitiateCall(c.Context(), callerID, calleeID, req.CallType)
	if err != nil {
		return response.InternalError(c, err)
	}

	// Генерируем токен для caller
	token, err := h.service.GenerateCallToken(call.RoomName, callerID, username)
	if err != nil {
		return response.InternalError(c, err)
	}

	// Отправляем push-уведомление callee
	// TODO: Implement push notification service

	return response.Success(c, fiber.Map{
		"call":      call,
		"token":     token,
		"room_name": call.RoomName,
		"ws_url":    h.service.GetLiveKitURL(),
	})
}

// AnswerCall - отвечаем на звонок
func (h *Handler) AnswerCall(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	username := c.Locals("username").(string)

	var req AnswerCallRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	uid, _ := uuid.Parse(userID)
	callID, _ := uuid.Parse(req.CallID)

	// Обновляем статус звонка
	call, err := h.service.AnswerCall(c.Context(), callID, uid)
	if err != nil {
		return response.InternalError(c, err)
	}

	// Генерируем токен для callee
	token, err := h.service.GenerateCallToken(call.RoomName, uid, username)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"call":      call,
		"token":     token,
		"room_name": call.RoomName,
		"ws_url":    h.service.GetLiveKitURL(),
	})
}

// RejectCall - отклоняем звонок
func (h *Handler) RejectCall(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req RejectCallRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	uid, _ := uuid.Parse(userID)
	callID, _ := uuid.Parse(req.CallID)

	if err := h.service.RejectCall(c.Context(), callID, uid); err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Call rejected",
	})
}

// EndCall - завершаем звонок
func (h *Handler) EndCall(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req EndCallRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	uid, _ := uuid.Parse(userID)
	callID, _ := uuid.Parse(req.CallID)

	call, err := h.service.EndCall(c.Context(), callID, uid)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, call)
}

// HandleWebSocket обрабатывает WebSocket соединения для сигналинга звонков
func (h *Handler) HandleWebSocket(c *websocket.Conn) {
	// Получаем userID из query параметров или заголовков
	userID := c.Query("user_id")
	if userID == "" {
		c.WriteMessage(websocket.TextMessage, []byte(`{"error":"user_id required"}`))
		c.Close()
		return
	}

	// Преобразуем в UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid user_id"}`))
		c.Close()
		return
	}

	// Логируем подключение
	log.Printf("WebSocket connected: %s", userID)

	// Основной цикл обработки сообщений
	for {
		messageType, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Эхо-ответ для тестирования
		if messageType == websocket.TextMessage {
			log.Printf("Received from %s: %s", uid, string(msg))

			// Отправляем обратно для подтверждения
			response := map[string]interface{}{
				"type": "pong",
				"data": string(msg),
			}

			if err := c.WriteJSON(response); err != nil {
				log.Printf("Write error: %v", err)
				break
			}
		}
	}

	log.Printf("WebSocket disconnected: %s", userID)
}

// GetCallHistory - история звонков
func (h *Handler) GetCallHistory(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, _ := uuid.Parse(userID)

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	calls, err := h.service.GetCallHistory(c.Context(), uid, limit, offset)
	if err != nil {
		return response.InternalError(c, err)
	}

	return response.Success(c, calls)
}

// Request DTOs
type TokenRequest struct {
	RoomName string `json:"room_name" validate:"required"`
}

type InitiateCallRequest struct {
	CalleeID string `json:"callee_id" validate:"required,uuid"`
	CallType string `json:"call_type" validate:"required,oneof=audio video"`
}

type AnswerCallRequest struct {
	CallID string `json:"call_id" validate:"required,uuid"`
}

type RejectCallRequest struct {
	CallID string `json:"call_id" validate:"required,uuid"`
}

type EndCallRequest struct {
	CallID string `json:"call_id" validate:"required,uuid"`
}
