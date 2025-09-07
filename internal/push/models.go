package push

import (
	"time"

	"github.com/google/uuid"
)

// DeviceToken представляет токен устройства для push уведомлений
type DeviceToken struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Token        string    `json:"token"`
	DeviceType   string    `json:"device_type"` // "ios", "android"
	PushType     string    `json:"push_type"`   // "fcm", "apns", "voip"
	DeviceInfo   string    `json:"device_info,omitempty"`
	AppVersion   string    `json:"app_version,omitempty"`
	IsActive     bool      `json:"is_active"`
	LastUsedAt   time.Time `json:"last_used_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// PushNotification представляет push уведомление
type PushNotification struct {
	Type     string                 `json:"type"`     // "call_incoming", "call_missed", "call_ended"
	Priority string                 `json:"priority"` // "high", "normal"
	Data     map[string]interface{} `json:"data"`
}

// CallPushData данные для входящего звонка
type CallPushData struct {
	CallID     string `json:"call_id"`
	CallerID   string `json:"caller_id"`
	CallerName string `json:"caller_name"`
	CallType   string `json:"call_type"` // "audio", "video"
	RoomName   string `json:"room_name"`
	Token      string `json:"token,omitempty"` // LiveKit token for direct connection
}

// FCMMessage структура для Firebase Cloud Messaging
type FCMMessage struct {
	To   string `json:"to"`
	Data struct {
		Type       string `json:"type"`
		CallID     string `json:"call_id"`
		CallerID   string `json:"caller_id"`
		CallerName string `json:"caller_name"`
		CallType   string `json:"call_type"`
		RoomName   string `json:"room_name"`
		Token      string `json:"token,omitempty"`
	} `json:"data"`
	Priority    string `json:"priority"`
	ContentAvailable bool `json:"content_available"`
	TimeToLive  int    `json:"time_to_live"`
}

// APNsMessage структура для Apple Push Notification Service
type APNsMessage struct {
	DeviceToken string `json:"device_token"`
	Payload     struct {
		APS struct {
			Alert struct {
				Title string `json:"title"`
				Body  string `json:"body"`
			} `json:"alert"`
			Badge            int  `json:"badge,omitempty"`
			Sound            string `json:"sound"`
			ContentAvailable int  `json:"content-available"`
		} `json:"aps"`
		CallID     string `json:"call_id"`
		CallerID   string `json:"caller_id"`
		CallerName string `json:"caller_name"`
		CallType   string `json:"call_type"`
		RoomName   string `json:"room_name"`
		Token      string `json:"token,omitempty"`
	} `json:"payload"`
}

// VoIPPushMessage структура для VoIP push notifications (iOS)
type VoIPPushMessage struct {
	DeviceToken string `json:"device_token"`
	Payload     struct {
		CallID     string `json:"call_id"`
		CallerID   string `json:"caller_id"`
		CallerName string `json:"caller_name"`
		CallType   string `json:"call_type"`
		RoomName   string `json:"room_name"`
		Token      string `json:"token,omitempty"`
	} `json:"payload"`
}

// RegisterTokenRequest запрос на регистрацию токена
type RegisterTokenRequest struct {
	Token      string `json:"token" validate:"required"`
	DeviceType string `json:"device_type" validate:"required,oneof=ios android"`
	PushType   string `json:"push_type" validate:"required,oneof=fcm apns voip"`
	DeviceInfo string `json:"device_info,omitempty"`
	AppVersion string `json:"app_version,omitempty"`
}