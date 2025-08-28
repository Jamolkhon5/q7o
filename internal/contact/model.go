package contact

import (
	"github.com/google/uuid"
	"time"
)

type Contact struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	ContactID  uuid.UUID  `json:"contact_id"`
	LastCallAt *time.Time `json:"last_call_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ContactRequest struct {
	ID          uuid.UUID  `json:"id"`
	SenderID    uuid.UUID  `json:"sender_id"`
	ReceiverID  uuid.UUID  `json:"receiver_id"`
	Status      string     `json:"status"` // pending, accepted, rejected
	Message     *string    `json:"message,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
}

type ContactWithUser struct {
	ID         uuid.UUID  `json:"id"`
	ContactID  uuid.UUID  `json:"contact_id"`
	Username   string     `json:"username"`
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	Email      string     `json:"email"`
	AvatarURL  *string    `json:"avatar_url"`
	Status     string     `json:"status"`
	LastCallAt *time.Time `json:"last_call_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ContactRequestWithUser struct {
	ID          uuid.UUID  `json:"id"`
	SenderID    uuid.UUID  `json:"sender_id"`
	ReceiverID  uuid.UUID  `json:"receiver_id"`
	Username    string     `json:"username"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	AvatarURL   *string    `json:"avatar_url"`
	Status      string     `json:"status"`
	Message     *string    `json:"message,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
	RequestType string     `json:"request_type"` // incoming or outgoing
}

type SendContactRequestDTO struct {
	ReceiverID string  `json:"receiver_id" validate:"required,uuid"`
	Message    *string `json:"message" validate:"omitempty,max=500"`
}

type RespondToRequestDTO struct {
	RequestID string `json:"request_id" validate:"required,uuid"`
	Accept    bool   `json:"accept" validate:"required"`
}
