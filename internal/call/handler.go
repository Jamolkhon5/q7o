package call

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"log"
	"q7o/internal/common/response"
	"q7o/internal/push"
)

type Handler struct {
	service *Service
	wsHub   *WSHub
}

func NewHandler(service *Service, wsHub *WSHub) *Handler {
	return &Handler{
		service: service,
		wsHub:   wsHub,
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

// InitiateCall создает новый звонок и отправляет WebSocket сигнал
func (h *Handler) InitiateCall(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req InitiateCallRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	callerID, _ := uuid.Parse(userID)
	calleeID, _ := uuid.Parse(req.CalleeID)

	// Создаем запись о звонке и получаем токены для обеих сторон
	call, callerToken, calleeToken, err := h.service.InitiateCall(c.Context(), callerID, calleeID, req.CallType)
	if err != nil {
		return response.InternalError(c, err)
	}

	// Отправляем WebSocket сигнал получателю о входящем звонке
	signalData := map[string]interface{}{
		"call_id":     call.ID.String(),
		"room_name":   call.RoomName,
		"caller_name": call.CallerName,
		"callee_name": call.CalleeName,
	}
	signalJSON, _ := json.Marshal(signalData)

	signal := &CallSignal{
		Type:       "ring",
		FromID:     callerID,
		ToID:       calleeID,
		RoomName:   call.RoomName,
		CallType:   req.CallType,
		CallID:     call.ID.String(),
		CallerName: call.CallerName,
		CalleeName: call.CalleeName,
		Data:       signalJSON,
	}

	// Отправляем сигнал через WebSocket Hub
	h.wsHub.broadcast <- signal

	log.Printf("Sent ring signal from %s to %s for call %s", callerID, calleeID, call.ID)

	// 🚀 КРИТИЧЕСКИ ВАЖНО: Отправляем push уведомление для фонового режима
	// Это позволит получать звонки даже когда приложение закрыто
	if h.service.pushService != nil {
		go func() {
			pushData := &push.CallPushData{
				CallID:     call.ID.String(),
				CallerID:   call.CallerID.String(),
				CallerName: call.CallerName,
				CallType:   call.CallType,
				RoomName:   call.RoomName,
				Token:      calleeToken, // Передаем токен для прямого подключения
			}

			if err := h.service.pushService.SendCallNotification(c.Context(), call.CalleeID, pushData); err != nil {
				log.Printf("Failed to send push notification for call %s: %v", call.ID, err)
			} else {
				log.Printf("Push notification sent successfully for call %s", call.ID)
			}
		}()
	}

	// Возвращаем токен звонящему сразу
	return response.Success(c, fiber.Map{
		"call":      call,
		"token":     callerToken,
		"room_name": call.RoomName,
		"ws_url":    h.service.GetLiveKitURL(),
	})
}

// AnswerCall - отвечаем на звонок и уведомляем звонящего
func (h *Handler) AnswerCall(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req AnswerCallRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	uid, _ := uuid.Parse(userID)
	callID, _ := uuid.Parse(req.CallID)

	// Обновляем статус звонка и получаем токен для callee
	call, calleeToken, err := h.service.AnswerCall(c.Context(), callID, uid)
	if err != nil {
		return response.InternalError(c, err)
	}

	// Отправляем сигнал звонящему что на звонок ответили
	signal := &CallSignal{
		Type:     "answered",
		FromID:   call.CalleeID,
		ToID:     call.CallerID,
		RoomName: call.RoomName,
		CallID:   call.ID.String(),
		CallType: call.CallType,
	}
	h.wsHub.broadcast <- signal

	log.Printf("Call %s answered by %s", call.ID, uid)

	// Возвращаем токен для получателя
	return response.Success(c, fiber.Map{
		"call":      call,
		"token":     calleeToken,
		"room_name": call.RoomName,
		"ws_url":    h.service.GetLiveKitURL(),
	})
}

// RejectCall - отклоняем звонок и уведомляем звонящего
func (h *Handler) RejectCall(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var req RejectCallRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	uid, _ := uuid.Parse(userID)
	callID, _ := uuid.Parse(req.CallID)

	// Получаем информацию о звонке перед отклонением
	call, err := h.service.GetCall(c.Context(), callID)
	if err != nil {
		return response.InternalError(c, err)
	}

	if err := h.service.RejectCall(c.Context(), callID, uid); err != nil {
		return response.InternalError(c, err)
	}

	// Отправляем сигнал звонящему что звонок отклонен
	signal := &CallSignal{
		Type:   "rejected",
		FromID: call.CalleeID,
		ToID:   call.CallerID,
		CallID: call.ID.String(),
	}
	h.wsHub.broadcast <- signal

	log.Printf("Call %s rejected by %s", call.ID, uid)

	return response.Success(c, fiber.Map{
		"message": "Call rejected",
	})
}

// EndCall - завершаем звонок и уведомляем участников
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

	// Определяем кому отправить уведомление
	var toID uuid.UUID
	if call.CallerID == uid {
		toID = call.CalleeID
	} else {
		toID = call.CallerID
	}

	// Отправляем сигнал другому участнику что звонок завершен
	signal := &CallSignal{
		Type:   "ended",
		FromID: uid,
		ToID:   toID,
		CallID: call.ID.String(),
	}
	h.wsHub.broadcast <- signal

	log.Printf("Call %s ended by %s", call.ID, uid)

	return response.Success(c, call)
}

// HandleWebSocket обрабатывает WebSocket соединения для сигналинга звонков
// HandleWebSocket обрабатывает WebSocket соединения для сигналинга звонков
func (h *Handler) HandleWebSocket(c *websocket.Conn, hub *WSHub) {
	// Получаем userID и token из query параметров
	userID := c.Query("user_id")
	token := c.Query("token")

	if userID == "" {
		c.WriteMessage(websocket.TextMessage, []byte(`{"error":"user_id required"}`))
		c.Close()
		return
	}

	if token == "" {
		c.WriteMessage(websocket.TextMessage, []byte(`{"error":"token required"}`))
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

	// Проверяем токен через сервис auth
	valid, tokenUID, err := h.service.ValidateToken(token)
	if err != nil || !valid {
		c.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid token"}`))
		c.Close()
		return
	}

	// Проверяем что userID соответствует токену
	if tokenUID.String() != userID {
		c.WriteMessage(websocket.TextMessage, []byte(`{"error":"user_id mismatch"}`))
		c.Close()
		return
	}

	// Регистрируем клиента в хабе
	client := &Client{
		ID:   uid,
		Conn: c,
	}
	hub.register <- client

	// Проверяем и отправляем офлайн сигналы если есть
	if offlineSignals, err := hub.GetOfflineSignals(uid); err == nil && len(offlineSignals) > 0 {
		for _, signal := range offlineSignals {
			if err := c.WriteJSON(signal); err != nil {
				log.Printf("Error sending offline signal: %v", err)
			}
		}
	}

	// Отправляем подтверждение подключения
	c.WriteJSON(map[string]interface{}{
		"type":    "connected",
		"user_id": userID,
	})

	log.Printf("WebSocket connected: %s", userID)

	// Основной цикл обработки сообщений
	defer func() {
		hub.unregister <- client
		c.Close()
		log.Printf("WebSocket disconnected: %s", userID)
	}()

	for {
		messageType, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Обработка входящих сообщений от клиента
		if messageType == websocket.TextMessage {
			log.Printf("Received from %s: %s", uid, string(msg))

			var signal CallSignal
			if err := json.Unmarshal(msg, &signal); err == nil {
				// ИСПРАВЛЕНИЕ: Игнорируем служебные сигналы
				if signal.Type == "connected" {
					// Не пересылаем connected сигнал, это только для подтверждения соединения
					continue
				}

				signal.FromID = uid
				// Пересылаем только реальные сигналы звонков и контактов
				hub.broadcast <- &signal
			}
		}
	}
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
