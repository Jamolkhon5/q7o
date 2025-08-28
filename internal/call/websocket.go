package call

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CallSignal struct {
	Type       string          `json:"type"` // offer, answer, ice-candidate, ring, hangup, answered, rejected, ended, missed, contact_request_received, contact_request_accepted, contact_request_rejected, contact_removed
	FromID     uuid.UUID       `json:"from_id"`
	ToID       uuid.UUID       `json:"to_id"`
	RoomName   string          `json:"room_name,omitempty"`
	CallType   string          `json:"call_type,omitempty"` // audio, video
	CallID     string          `json:"call_id,omitempty"`
	CallerName string          `json:"caller_name,omitempty"`
	CalleeName string          `json:"callee_name,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

type WSHub struct {
	clients    map[uuid.UUID]*websocket.Conn
	register   chan *Client
	unregister chan *Client
	broadcast  chan *CallSignal
	redis      *redis.Client
}

type Client struct {
	ID   uuid.UUID
	Conn *websocket.Conn
}

func NewWSHub(redis *redis.Client) *WSHub {
	return &WSHub{
		clients:    make(map[uuid.UUID]*websocket.Conn),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *CallSignal),
		redis:      redis,
	}
}

// Broadcast returns the broadcast channel for sending signals
func (h *WSHub) Broadcast() chan<- *CallSignal {
	return h.broadcast
}

func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.ID] = client.Conn
			log.Printf("Client %s connected", client.ID)

		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				log.Printf("Client %s disconnected", client.ID)
			}

		case signal := <-h.broadcast:
			// Send to specific client
			if conn, ok := h.clients[signal.ToID]; ok {
				if err := conn.WriteJSON(signal); err != nil {
					log.Printf("Error sending signal: %v", err)
					conn.Close()
					delete(h.clients, signal.ToID)
				} else {
					log.Printf("Sent %s signal to %s", signal.Type, signal.ToID)
				}
			} else {
				// Store in Redis for offline delivery
				h.storeOfflineSignal(signal)
				log.Printf("Stored offline signal for %s", signal.ToID)
			}
		}
	}
}

func (h *WSHub) storeOfflineSignal(signal *CallSignal) {
	ctx := context.Background()
	data, _ := json.Marshal(signal)
	key := "offline_signal:" + signal.ToID.String()

	// Store longer for contact notifications than call signals
	expiration := 30 * time.Second
	if signal.Type == "contact_request_received" ||
		signal.Type == "contact_request_accepted" ||
		signal.Type == "contact_request_rejected" ||
		signal.Type == "contact_removed" {
		expiration = 24 * time.Hour // Keep contact notifications for 24 hours
	}

	h.redis.LPush(ctx, key, data)
	h.redis.Expire(ctx, key, expiration)
}

func (h *WSHub) GetOfflineSignals(userID uuid.UUID) ([]*CallSignal, error) {
	ctx := context.Background()
	key := "offline_signal:" + userID.String()

	data, err := h.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var signals []*CallSignal
	for _, d := range data {
		var signal CallSignal
		if err := json.Unmarshal([]byte(d), &signal); err == nil {
			signals = append(signals, &signal)
		}
	}

	// Clear after retrieval
	h.redis.Del(ctx, key)

	return signals, nil
}
