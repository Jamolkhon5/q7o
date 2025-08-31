package settings

import (
	"github.com/google/uuid"
	"time"
)

type UserSettings struct {
	ID                   uuid.UUID `json:"id"`
	UserID               uuid.UUID `json:"user_id"`
	NotificationsCall    bool      `json:"notifications_call"`
	NotificationsMeeting bool      `json:"notifications_meeting"`
	NotificationsChat    bool      `json:"notifications_chat"`
	Privacy              string    `json:"privacy" validate:"oneof=public friends private"` // public, friends, private
	Theme                string    `json:"theme" validate:"oneof=light dark auto"`          // light, dark, auto
	Language             string    `json:"language" validate:"oneof=en ru zh"`              // en, ru, zh - ДОБАВЛЕН zh
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type UpdateSettingsDTO struct {
	NotificationsCall    *bool   `json:"notifications_call"`
	NotificationsMeeting *bool   `json:"notifications_meeting"`
	NotificationsChat    *bool   `json:"notifications_chat"`
	Privacy              *string `json:"privacy" validate:"omitempty,oneof=public friends private"`
	Theme                *string `json:"theme" validate:"omitempty,oneof=light dark auto"`
	Language             *string `json:"language" validate:"omitempty,oneof=en ru zh"` // ДОБАВЛЕН zh
}

type SettingsResponse struct {
	ID                   uuid.UUID `json:"id"`
	UserID               uuid.UUID `json:"user_id"`
	NotificationsCall    bool      `json:"notifications_call"`
	NotificationsMeeting bool      `json:"notifications_meeting"`
	NotificationsChat    bool      `json:"notifications_chat"`
	Privacy              string    `json:"privacy"`
	Theme                string    `json:"theme"`
	Language             string    `json:"language"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Convert UserSettings to SettingsResponse
func (s *UserSettings) ToResponse() *SettingsResponse {
	return &SettingsResponse{
		ID:                   s.ID,
		UserID:               s.UserID,
		NotificationsCall:    s.NotificationsCall,
		NotificationsMeeting: s.NotificationsMeeting,
		NotificationsChat:    s.NotificationsChat,
		Privacy:              s.Privacy,
		Theme:                s.Theme,
		Language:             s.Language,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}
