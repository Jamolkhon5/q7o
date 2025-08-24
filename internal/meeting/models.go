package meeting

import (
	"time"

	"github.com/google/uuid"
)

type Meeting struct {
	ID               uuid.UUID  `json:"id"`
	MeetingCode      string     `json:"meeting_code"`
	RoomName         string     `json:"room_name"`
	HostID           uuid.UUID  `json:"host_id"`
	HostName         string     `json:"host_name,omitempty"`
	Title            string     `json:"title"`
	Description      string     `json:"description,omitempty"`
	MeetingType      string     `json:"meeting_type"`
	ScheduledAt      *time.Time `json:"scheduled_at,omitempty"`
	MaxParticipants  int        `json:"max_participants"`
	IsActive         bool       `json:"is_active"`
	RequiresAuth     bool       `json:"requires_auth"`
	AllowGuests      bool       `json:"allow_guests"`
	CreatedAt        time.Time  `json:"created_at"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
	ParticipantCount int        `json:"participant_count,omitempty"`
}

type MeetingParticipant struct {
	ID                uuid.UUID  `json:"id"`
	MeetingID         uuid.UUID  `json:"meeting_id"`
	UserID            *uuid.UUID `json:"user_id,omitempty"`
	GuestName         string     `json:"guest_name,omitempty"`
	DisplayName       string     `json:"display_name"`
	ParticipantRole   string     `json:"participant_role"`
	JoinedAt          time.Time  `json:"joined_at"`
	LeftAt            *time.Time `json:"left_at,omitempty"`
	IsActive          bool       `json:"is_active"`
	AudioEnabled      bool       `json:"audio_enabled"`
	VideoEnabled      bool       `json:"video_enabled"`
	ScreenSharing     bool       `json:"screen_sharing"`
	ConnectionQuality string     `json:"connection_quality,omitempty"`
}

type CreateMeetingRequest struct {
	Title        string     `json:"title"`
	Description  string     `json:"description,omitempty"`
	MeetingType  string     `json:"meeting_type,omitempty"`
	ScheduledAt  *time.Time `json:"scheduled_at,omitempty"`
	RequiresAuth bool       `json:"requires_auth"`
	AllowGuests  bool       `json:"allow_guests"`
}

type JoinMeetingRequest struct {
	MeetingCode  string `json:"meeting_code" validate:"required"`
	GuestName    string `json:"guest_name,omitempty"`
	AudioEnabled bool   `json:"audio_enabled"`
	VideoEnabled bool   `json:"video_enabled"`
}

type PreJoinCheckRequest struct {
	MeetingCode string `json:"meeting_code" validate:"required"`
}

type PreJoinCheckResponse struct {
	Valid            bool     `json:"valid"`
	Meeting          *Meeting `json:"meeting,omitempty"`
	RequiresAuth     bool     `json:"requires_auth"`
	ParticipantCount int      `json:"participant_count"`
	HostName         string   `json:"host_name,omitempty"`
}

type MeetingTokenResponse struct {
	Token   string  `json:"token"`
	Meeting Meeting `json:"meeting"`
	WsUrl   string  `json:"ws_url"`
	Role    string  `json:"role"`
}

type UpdateParticipantRequest struct {
	AudioEnabled  *bool `json:"audio_enabled,omitempty"`
	VideoEnabled  *bool `json:"video_enabled,omitempty"`
	ScreenSharing *bool `json:"screen_sharing,omitempty"`
}
