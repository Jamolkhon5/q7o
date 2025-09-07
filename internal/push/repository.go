package push

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// RegisterDeviceToken регистрирует или обновляет токен устройства
func (r *Repository) RegisterDeviceToken(ctx context.Context, token *DeviceToken) error {
	query := `
		INSERT INTO device_tokens (
			id, user_id, token, device_type, push_type, device_info, 
			app_version, is_active, last_used_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, token, device_type) 
		DO UPDATE SET 
			push_type = EXCLUDED.push_type,
			device_info = EXCLUDED.device_info,
			app_version = EXCLUDED.app_version,
			is_active = EXCLUDED.is_active,
			last_used_at = EXCLUDED.last_used_at
	`

	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.Token, token.DeviceType, token.PushType,
		token.DeviceInfo, token.AppVersion, token.IsActive, token.LastUsedAt, token.CreatedAt,
	)

	return err
}

// GetActiveTokensForUser получает активные токены для пользователя
func (r *Repository) GetActiveTokensForUser(ctx context.Context, userID uuid.UUID) ([]*DeviceToken, error) {
	query := `
		SELECT id, user_id, token, device_type, push_type, device_info, 
			   app_version, is_active, last_used_at, created_at
		FROM device_tokens 
		WHERE user_id = $1 AND is_active = true
		ORDER BY last_used_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*DeviceToken
	for rows.Next() {
		token := &DeviceToken{}
		err := rows.Scan(
			&token.ID, &token.UserID, &token.Token, &token.DeviceType, &token.PushType,
			&token.DeviceInfo, &token.AppVersion, &token.IsActive, &token.LastUsedAt, &token.CreatedAt,
		)
		if err != nil {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// GetTokensByType получает токены определенного типа для пользователя
func (r *Repository) GetTokensByType(ctx context.Context, userID uuid.UUID, pushType string) ([]*DeviceToken, error) {
	query := `
		SELECT id, user_id, token, device_type, push_type, device_info, 
			   app_version, is_active, last_used_at, created_at
		FROM device_tokens 
		WHERE user_id = $1 AND push_type = $2 AND is_active = true
		ORDER BY last_used_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, pushType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*DeviceToken
	for rows.Next() {
		token := &DeviceToken{}
		err := rows.Scan(
			&token.ID, &token.UserID, &token.Token, &token.DeviceType, &token.PushType,
			&token.DeviceInfo, &token.AppVersion, &token.IsActive, &token.LastUsedAt, &token.CreatedAt,
		)
		if err != nil {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

// DeactivateToken деактивирует токен
func (r *Repository) DeactivateToken(ctx context.Context, userID uuid.UUID, token string) error {
	query := `
		UPDATE device_tokens 
		SET is_active = false 
		WHERE user_id = $1 AND token = $2
	`

	_, err := r.db.ExecContext(ctx, query, userID, token)
	return err
}

// UpdateTokenUsage обновляет время последнего использования токена
func (r *Repository) UpdateTokenUsage(ctx context.Context, token string) error {
	query := `
		UPDATE device_tokens 
		SET last_used_at = $1 
		WHERE token = $2 AND is_active = true
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), token)
	return err
}

// CleanupOldTokens удаляет старые неактивные токены
func (r *Repository) CleanupOldTokens(ctx context.Context, olderThan time.Time) error {
	query := `
		DELETE FROM device_tokens 
		WHERE is_active = false AND created_at < $1
	`

	_, err := r.db.ExecContext(ctx, query, olderThan)
	return err
}