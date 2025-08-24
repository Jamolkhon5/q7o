package meeting

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"q7o/config"
	"q7o/internal/user"
)

type Service struct {
	repo     *Repository
	userRepo *user.Repository
	livekit  *LiveKitService
	redis    *redis.Client
	cfg      config.LiveKitConfig
}

func NewService(
	repo *Repository,
	userRepo *user.Repository,
	cfg config.LiveKitConfig,
	redis *redis.Client,
) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
		livekit:  NewLiveKitService(cfg),
		redis:    redis,
		cfg:      cfg,
	}
}

// GenerateMeetingCode generates a unique meeting code in format xxx-xxxx-xxx
func (s *Service) GenerateMeetingCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 10)
	rand.Read(b)

	var code strings.Builder
	for i, v := range b {
		if i == 3 || i == 7 {
			code.WriteByte('-')
		}
		code.WriteByte(charset[v%byte(len(charset))])
	}

	return code.String()
}

// CreateMeeting creates a new meeting room
func (s *Service) CreateMeeting(ctx context.Context, hostID uuid.UUID, req *CreateMeetingRequest) (*Meeting, error) {
	// Get host info
	host, err := s.userRepo.FindByID(ctx, hostID)
	if err != nil {
		return nil, errors.New("host not found")
	}

	// Generate unique meeting code
	var meetingCode string
	for attempts := 0; attempts < 5; attempts++ {
		meetingCode = s.GenerateMeetingCode()

		// Check if code already exists
		existing, _ := s.repo.FindByCode(ctx, meetingCode)
		if existing == nil {
			break
		}
	}

	if meetingCode == "" {
		return nil, errors.New("failed to generate unique meeting code")
	}

	// Create meeting
	meeting := &Meeting{
		ID:              uuid.New(),
		MeetingCode:     meetingCode,
		RoomName:        fmt.Sprintf("meeting_%s", meetingCode),
		HostID:          hostID,
		HostName:        host.Username,
		Title:           req.Title,
		Description:     req.Description,
		MeetingType:     "instant",
		MaxParticipants: 100,
		IsActive:        true,
		RequiresAuth:    req.RequiresAuth,
		AllowGuests:     req.AllowGuests,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}

	if req.MeetingType != "" {
		meeting.MeetingType = req.MeetingType
	}

	if req.ScheduledAt != nil {
		meeting.ScheduledAt = req.ScheduledAt
		meeting.MeetingType = "scheduled"
	}

	// Save to database
	if err := s.repo.CreateMeeting(ctx, meeting); err != nil {
		return nil, err
	}

	// Cache meeting code mapping in Redis
	cacheKey := fmt.Sprintf("meeting:code:%s", meetingCode)
	s.redis.Set(ctx, cacheKey, meeting.ID.String(), 24*time.Hour)

	// Add host as first participant
	participant := &MeetingParticipant{
		ID:              uuid.New(),
		MeetingID:       meeting.ID,
		UserID:          &hostID,
		DisplayName:     host.Username,
		ParticipantRole: "host",
		JoinedAt:        time.Now(),
		IsActive:        false, // Will be active when actually joins
		AudioEnabled:    true,
		VideoEnabled:    true,
	}

	s.repo.AddParticipant(ctx, participant)

	return meeting, nil
}

// ValidateMeetingCode checks if a meeting code is valid and returns meeting info
func (s *Service) ValidateMeetingCode(ctx context.Context, code string) (*PreJoinCheckResponse, error) {
	meeting, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if meeting == nil {
		return &PreJoinCheckResponse{
			Valid: false,
		}, nil
	}

	// Get participant count
	participants, _ := s.repo.GetMeetingParticipants(ctx, meeting.ID)

	return &PreJoinCheckResponse{
		Valid:            true,
		Meeting:          meeting,
		RequiresAuth:     meeting.RequiresAuth,
		ParticipantCount: len(participants),
		HostName:         meeting.HostName,
	}, nil
}

// JoinMeeting handles participant joining a meeting
func (s *Service) JoinMeeting(ctx context.Context, userID *uuid.UUID, req *JoinMeetingRequest) (*MeetingTokenResponse, error) {
	// Find meeting by code
	meeting, err := s.repo.FindByCode(ctx, req.MeetingCode)
	if err != nil {
		return nil, err
	}

	if meeting == nil {
		return nil, errors.New("meeting not found")
	}

	// Check if meeting requires auth
	if meeting.RequiresAuth && userID == nil {
		return nil, errors.New("authentication required")
	}

	// Check if guests are allowed
	if !meeting.AllowGuests && userID == nil {
		return nil, errors.New("guests not allowed")
	}

	// Get participant count
	participants, _ := s.repo.GetMeetingParticipants(ctx, meeting.ID)
	if len(participants) >= meeting.MaxParticipants {
		return nil, errors.New("meeting is full")
	}

	// Determine participant role
	role := "participant"
	var displayName string

	if userID != nil && *userID == meeting.HostID {
		role = "host"
		user, _ := s.userRepo.FindByID(ctx, *userID)
		if user != nil {
			displayName = user.Username
		}
	} else if userID != nil {
		user, _ := s.userRepo.FindByID(ctx, *userID)
		if user != nil {
			displayName = user.Username
		}
	} else {
		displayName = req.GuestName
		if displayName == "" {
			displayName = "Guest"
		}
	}

	// Add or update participant
	participant := &MeetingParticipant{
		ID:              uuid.New(),
		MeetingID:       meeting.ID,
		UserID:          userID,
		GuestName:       req.GuestName,
		DisplayName:     displayName,
		ParticipantRole: role,
		JoinedAt:        time.Now(),
		IsActive:        true,
		AudioEnabled:    req.AudioEnabled,
		VideoEnabled:    req.VideoEnabled,
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return nil, err
	}

	// Generate LiveKit token
	identity := displayName
	if userID != nil {
		identity = userID.String()
	} else {
		identity = fmt.Sprintf("guest_%s", participant.ID.String()[:8])
	}

	token, err := s.livekit.GenerateMeetingToken(meeting.RoomName, identity, displayName, role)
	if err != nil {
		return nil, err
	}

	// Get LiveKit URL
	wsUrl := s.cfg.PublicHost
	if wsUrl == "" {
		wsUrl = s.cfg.Host
	}

	return &MeetingTokenResponse{
		Token:   token,
		Meeting: *meeting,
		WsUrl:   wsUrl,
		Role:    role,
	}, nil
}

// LeaveMeeting handles participant leaving
func (s *Service) LeaveMeeting(ctx context.Context, meetingID, userID uuid.UUID) error {
	// Mark participant as inactive
	if err := s.repo.RemoveParticipant(ctx, meetingID, userID); err != nil {
		return err
	}

	// Check if meeting should end (no active participants)
	participants, _ := s.repo.GetMeetingParticipants(ctx, meetingID)
	if len(participants) == 0 {
		// End the meeting
		return s.repo.EndMeeting(ctx, meetingID)
	}

	// Check if host left and transfer host role if needed
	meeting, _ := s.repo.FindByID(ctx, meetingID)
	if meeting != nil && meeting.HostID == userID && len(participants) > 0 {
		// Transfer host to first participant
		// This is simplified - you might want more complex logic
		fmt.Printf("Host left, should transfer host role\n")
	}

	return nil
}

// EndMeeting ends a meeting (only host can do this)
func (s *Service) EndMeeting(ctx context.Context, meetingID, userID uuid.UUID) error {
	meeting, err := s.repo.FindByID(ctx, meetingID)
	if err != nil {
		return err
	}

	// Check if user is host
	if meeting.HostID != userID {
		return errors.New("only host can end meeting")
	}

	return s.repo.EndMeeting(ctx, meetingID)
}

// GetMeetingParticipants returns current participants
func (s *Service) GetMeetingParticipants(ctx context.Context, meetingID uuid.UUID) ([]*MeetingParticipant, error) {
	return s.repo.GetMeetingParticipants(ctx, meetingID)
}

// UpdateParticipantStatus updates participant media status
func (s *Service) UpdateParticipantStatus(ctx context.Context, meetingID, userID uuid.UUID, req *UpdateParticipantRequest) error {
	return s.repo.UpdateParticipantStatus(ctx, meetingID, userID, req)
}

// GetUserMeetings returns user's meeting history
func (s *Service) GetUserMeetings(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Meeting, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetUserMeetings(ctx, userID, limit, offset)
}

// CleanupExpiredMeetings runs periodically to clean up expired meetings
func (s *Service) CleanupExpiredMeetings(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.repo.CleanupExpiredMeetings(ctx); err != nil {
				fmt.Printf("Error cleaning up expired meetings: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
