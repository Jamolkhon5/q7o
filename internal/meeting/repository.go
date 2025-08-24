package meeting

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateMeeting creates a new meeting
func (r *Repository) CreateMeeting(ctx context.Context, meeting *Meeting) error {
	query := `
		INSERT INTO meetings (
			id, meeting_code, room_name, host_id, title, description,
			meeting_type, scheduled_at, max_participants, is_active,
			requires_auth, allow_guests, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		meeting.ID, meeting.MeetingCode, meeting.RoomName, meeting.HostID,
		meeting.Title, meeting.Description, meeting.MeetingType,
		meeting.ScheduledAt, meeting.MaxParticipants, meeting.IsActive,
		meeting.RequiresAuth, meeting.AllowGuests, meeting.CreatedAt,
		meeting.ExpiresAt,
	)

	return err
}

// FindByCode finds a meeting by its code
func (r *Repository) FindByCode(ctx context.Context, code string) (*Meeting, error) {
	query := `
		SELECT m.id, m.meeting_code, m.room_name, m.host_id, m.title,
			   m.description, m.meeting_type, m.scheduled_at, m.max_participants,
			   m.is_active, m.requires_auth, m.allow_guests, m.created_at,
			   m.ended_at, m.expires_at, u.username as host_name,
			   (SELECT COUNT(*) FROM meeting_participants mp 
			    WHERE mp.meeting_id = m.id AND mp.is_active = true) as participant_count
		FROM meetings m
		LEFT JOIN users u ON m.host_id = u.id
		WHERE m.meeting_code = $1 AND m.is_active = true AND m.expires_at > NOW()
	`

	meeting := &Meeting{}
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&meeting.ID, &meeting.MeetingCode, &meeting.RoomName, &meeting.HostID,
		&meeting.Title, &meeting.Description, &meeting.MeetingType,
		&meeting.ScheduledAt, &meeting.MaxParticipants, &meeting.IsActive,
		&meeting.RequiresAuth, &meeting.AllowGuests, &meeting.CreatedAt,
		&meeting.EndedAt, &meeting.ExpiresAt, &meeting.HostName,
		&meeting.ParticipantCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return meeting, err
}

// FindByID finds a meeting by ID
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Meeting, error) {
	query := `
		SELECT m.id, m.meeting_code, m.room_name, m.host_id, m.title,
			   m.description, m.meeting_type, m.scheduled_at, m.max_participants,
			   m.is_active, m.requires_auth, m.allow_guests, m.created_at,
			   m.ended_at, m.expires_at, u.username as host_name
		FROM meetings m
		LEFT JOIN users u ON m.host_id = u.id
		WHERE m.id = $1
	`

	meeting := &Meeting{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&meeting.ID, &meeting.MeetingCode, &meeting.RoomName, &meeting.HostID,
		&meeting.Title, &meeting.Description, &meeting.MeetingType,
		&meeting.ScheduledAt, &meeting.MaxParticipants, &meeting.IsActive,
		&meeting.RequiresAuth, &meeting.AllowGuests, &meeting.CreatedAt,
		&meeting.EndedAt, &meeting.ExpiresAt, &meeting.HostName,
	)

	return meeting, err
}

// GetUserMeetings gets all meetings for a user
func (r *Repository) GetUserMeetings(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Meeting, error) {
	query := `
		SELECT DISTINCT m.id, m.meeting_code, m.room_name, m.host_id, m.title,
			   m.description, m.meeting_type, m.scheduled_at, m.max_participants,
			   m.is_active, m.requires_auth, m.allow_guests, m.created_at,
			   m.ended_at, m.expires_at,
			   (SELECT COUNT(*) FROM meeting_participants mp 
			    WHERE mp.meeting_id = m.id AND mp.is_active = true) as participant_count
		FROM meetings m
		LEFT JOIN meeting_participants mp ON m.id = mp.meeting_id
		WHERE (m.host_id = $1 OR mp.user_id = $1)
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []*Meeting
	for rows.Next() {
		meeting := &Meeting{}
		err := rows.Scan(
			&meeting.ID, &meeting.MeetingCode, &meeting.RoomName, &meeting.HostID,
			&meeting.Title, &meeting.Description, &meeting.MeetingType,
			&meeting.ScheduledAt, &meeting.MaxParticipants, &meeting.IsActive,
			&meeting.RequiresAuth, &meeting.AllowGuests, &meeting.CreatedAt,
			&meeting.EndedAt, &meeting.ExpiresAt, &meeting.ParticipantCount,
		)
		if err != nil {
			return nil, err
		}
		meetings = append(meetings, meeting)
	}

	return meetings, nil
}

// AddParticipant adds a participant to a meeting
func (r *Repository) AddParticipant(ctx context.Context, participant *MeetingParticipant) error {
	query := `
		INSERT INTO meeting_participants (
			id, meeting_id, user_id, guest_name, participant_role,
			joined_at, is_active, audio_enabled, video_enabled, screen_sharing
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (meeting_id, user_id) 
		DO UPDATE SET 
			is_active = true,
			joined_at = CURRENT_TIMESTAMP,
			left_at = NULL,
			audio_enabled = EXCLUDED.audio_enabled,
			video_enabled = EXCLUDED.video_enabled
	`

	_, err := r.db.ExecContext(ctx, query,
		participant.ID, participant.MeetingID, participant.UserID,
		participant.GuestName, participant.ParticipantRole, participant.JoinedAt,
		participant.IsActive, participant.AudioEnabled, participant.VideoEnabled,
		participant.ScreenSharing,
	)

	return err
}

// RemoveParticipant marks a participant as inactive
func (r *Repository) RemoveParticipant(ctx context.Context, meetingID, userID uuid.UUID) error {
	query := `
		UPDATE meeting_participants 
		SET is_active = false, left_at = CURRENT_TIMESTAMP
		WHERE meeting_id = $1 AND user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, meetingID, userID)
	return err
}

// GetMeetingParticipants gets all active participants in a meeting
func (r *Repository) GetMeetingParticipants(ctx context.Context, meetingID uuid.UUID) ([]*MeetingParticipant, error) {
	query := `
		SELECT mp.id, mp.meeting_id, mp.user_id, mp.guest_name,
			   COALESCE(u.username, mp.guest_name) as display_name,
			   mp.participant_role, mp.joined_at, mp.left_at,
			   mp.is_active, mp.audio_enabled, mp.video_enabled,
			   mp.screen_sharing, mp.connection_quality
		FROM meeting_participants mp
		LEFT JOIN users u ON mp.user_id = u.id
		WHERE mp.meeting_id = $1 AND mp.is_active = true
		ORDER BY mp.joined_at
	`

	rows, err := r.db.QueryContext(ctx, query, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*MeetingParticipant
	for rows.Next() {
		p := &MeetingParticipant{}
		err := rows.Scan(
			&p.ID, &p.MeetingID, &p.UserID, &p.GuestName, &p.DisplayName,
			&p.ParticipantRole, &p.JoinedAt, &p.LeftAt, &p.IsActive,
			&p.AudioEnabled, &p.VideoEnabled, &p.ScreenSharing,
			&p.ConnectionQuality,
		)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// UpdateParticipantStatus updates participant's audio/video/screen status
func (r *Repository) UpdateParticipantStatus(ctx context.Context, meetingID, userID uuid.UUID, req *UpdateParticipantRequest) error {
	updates := []string{}
	args := []interface{}{}
	argCount := 1

	if req.AudioEnabled != nil {
		updates = append(updates, fmt.Sprintf("audio_enabled = $%d", argCount))
		args = append(args, *req.AudioEnabled)
		argCount++
	}

	if req.VideoEnabled != nil {
		updates = append(updates, fmt.Sprintf("video_enabled = $%d", argCount))
		args = append(args, *req.VideoEnabled)
		argCount++
	}

	if req.ScreenSharing != nil {
		updates = append(updates, fmt.Sprintf("screen_sharing = $%d", argCount))
		args = append(args, *req.ScreenSharing)
		argCount++
	}

	if len(updates) == 0 {
		return nil
	}

	args = append(args, meetingID, userID)
	query := fmt.Sprintf(`
		UPDATE meeting_participants 
		SET %s
		WHERE meeting_id = $%d AND user_id = $%d
	`,
		strings.Join(updates, ", "),
		argCount, argCount+1,
	)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// EndMeeting marks a meeting as ended
func (r *Repository) EndMeeting(ctx context.Context, meetingID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Mark meeting as ended
	_, err = tx.ExecContext(ctx, `
		UPDATE meetings 
		SET is_active = false, ended_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, meetingID)
	if err != nil {
		return err
	}

	// Mark all participants as left
	_, err = tx.ExecContext(ctx, `
		UPDATE meeting_participants
		SET is_active = false, left_at = CURRENT_TIMESTAMP
		WHERE meeting_id = $1 AND is_active = true
	`, meetingID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// CleanupExpiredMeetings removes expired meetings
func (r *Repository) CleanupExpiredMeetings(ctx context.Context) error {
	query := `
		UPDATE meetings 
		SET is_active = false, ended_at = CURRENT_TIMESTAMP
		WHERE is_active = true AND expires_at < NOW()
	`

	_, err := r.db.ExecContext(ctx, query)
	return err
}
