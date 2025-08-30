package settings

import (
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type Repository interface {
	GetByUserID(userID uuid.UUID) (*UserSettings, error)
	Create(userID uuid.UUID) (*UserSettings, error)
	Update(userID uuid.UUID, dto *UpdateSettingsDTO) (*UserSettings, error)
	Delete(userID uuid.UUID) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetByUserID(userID uuid.UUID) (*UserSettings, error) {
	query := `
		SELECT id, user_id, notifications_call, notifications_meeting, notifications_chat, 
		       privacy, theme, language, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1`

	settings := &UserSettings{}
	err := r.db.QueryRow(query, userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.NotificationsCall,
		&settings.NotificationsMeeting,
		&settings.NotificationsChat,
		&settings.Privacy,
		&settings.Theme,
		&settings.Language,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// If settings don't exist, create default ones
			return r.Create(userID)
		}
		return nil, err
	}

	return settings, nil
}

func (r *repository) Create(userID uuid.UUID) (*UserSettings, error) {
	settings := &UserSettings{
		ID:                   uuid.New(),
		UserID:               userID,
		NotificationsCall:    true,
		NotificationsMeeting: true,
		NotificationsChat:    true,
		Privacy:              "friends",
		Theme:                "auto",
		Language:             "en",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	query := `
		INSERT INTO user_settings (id, user_id, notifications_call, notifications_meeting, 
		                          notifications_chat, privacy, theme, language, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, user_id, notifications_call, notifications_meeting, notifications_chat, 
		          privacy, theme, language, created_at, updated_at`

	err := r.db.QueryRow(
		query,
		settings.ID,
		settings.UserID,
		settings.NotificationsCall,
		settings.NotificationsMeeting,
		settings.NotificationsChat,
		settings.Privacy,
		settings.Theme,
		settings.Language,
		settings.CreatedAt,
		settings.UpdatedAt,
	).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.NotificationsCall,
		&settings.NotificationsMeeting,
		&settings.NotificationsChat,
		&settings.Privacy,
		&settings.Theme,
		&settings.Language,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return settings, nil
}

func (r *repository) Update(userID uuid.UUID, dto *UpdateSettingsDTO) (*UserSettings, error) {
	// Get current settings
	currentSettings, err := r.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Update only provided fields
	if dto.NotificationsCall != nil {
		currentSettings.NotificationsCall = *dto.NotificationsCall
	}
	if dto.NotificationsMeeting != nil {
		currentSettings.NotificationsMeeting = *dto.NotificationsMeeting
	}
	if dto.NotificationsChat != nil {
		currentSettings.NotificationsChat = *dto.NotificationsChat
	}
	if dto.Privacy != nil {
		currentSettings.Privacy = *dto.Privacy
	}
	if dto.Theme != nil {
		currentSettings.Theme = *dto.Theme
	}
	if dto.Language != nil {
		currentSettings.Language = *dto.Language
	}

	currentSettings.UpdatedAt = time.Now()

	query := `
		UPDATE user_settings 
		SET notifications_call = $2, notifications_meeting = $3, notifications_chat = $4,
		    privacy = $5, theme = $6, language = $7, updated_at = $8
		WHERE user_id = $1
		RETURNING id, user_id, notifications_call, notifications_meeting, notifications_chat, 
		          privacy, theme, language, created_at, updated_at`

	err = r.db.QueryRow(
		query,
		userID,
		currentSettings.NotificationsCall,
		currentSettings.NotificationsMeeting,
		currentSettings.NotificationsChat,
		currentSettings.Privacy,
		currentSettings.Theme,
		currentSettings.Language,
		currentSettings.UpdatedAt,
	).Scan(
		&currentSettings.ID,
		&currentSettings.UserID,
		&currentSettings.NotificationsCall,
		&currentSettings.NotificationsMeeting,
		&currentSettings.NotificationsChat,
		&currentSettings.Privacy,
		&currentSettings.Theme,
		&currentSettings.Language,
		&currentSettings.CreatedAt,
		&currentSettings.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return currentSettings, nil
}

func (r *repository) Delete(userID uuid.UUID) error {
	query := `DELETE FROM user_settings WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}