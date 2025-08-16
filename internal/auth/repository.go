package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db    *sql.DB
	redis *redis.Client
}

func NewRepository(db *sql.DB, redis *redis.Client) *Repository {
	return &Repository{
		db:    db,
		redis: redis,
	}
}

func (r *Repository) SaveRefreshToken(ctx context.Context, userID uuid.UUID, token, deviceInfo string) error {
	query := `
        INSERT INTO sessions (user_id, refresh_token, device_info, expires_at)
        VALUES ($1, $2, $3, $4)
    `
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err := r.db.ExecContext(ctx, query, userID, token, deviceInfo, expiresAt)

	// Also save in Redis for fast lookup
	key := fmt.Sprintf("refresh:%s", token)
	data := map[string]string{
		"user_id": userID.String(),
		"token":   token,
	}
	jsonData, _ := json.Marshal(data)
	r.redis.Set(ctx, key, jsonData, 7*24*time.Hour)

	return err
}

func (r *Repository) RefreshTokenExists(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	// Check Redis first
	key := fmt.Sprintf("refresh:%s", token)
	exists := r.redis.Exists(ctx, key).Val()
	if exists > 0 {
		return true, nil
	}

	// Fallback to database
	query := `
        SELECT COUNT(*) FROM sessions 
        WHERE user_id = $1 AND refresh_token = $2 AND expires_at > NOW()
    `
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, token).Scan(&count)
	return count > 0, err
}

func (r *Repository) UpdateRefreshToken(ctx context.Context, userID uuid.UUID, oldToken, newToken string) error {
	// Delete old token from Redis
	oldKey := fmt.Sprintf("refresh:%s", oldToken)
	r.redis.Del(ctx, oldKey)

	// Update in database
	query := `
        UPDATE sessions 
        SET refresh_token = $1, expires_at = $2
        WHERE user_id = $3 AND refresh_token = $4
    `
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err := r.db.ExecContext(ctx, query, newToken, expiresAt, userID, oldToken)

	// Save new token in Redis
	newKey := fmt.Sprintf("refresh:%s", newToken)
	data := map[string]string{
		"user_id": userID.String(),
		"token":   newToken,
	}
	jsonData, _ := json.Marshal(data)
	r.redis.Set(ctx, newKey, jsonData, 7*24*time.Hour)

	return err
}

func (r *Repository) DeleteRefreshToken(ctx context.Context, userID uuid.UUID, token string) error {
	// Delete from Redis
	key := fmt.Sprintf("refresh:%s", token)
	r.redis.Del(ctx, key)

	// Delete from database
	query := `DELETE FROM sessions WHERE user_id = $1 AND refresh_token = $2`
	_, err := r.db.ExecContext(ctx, query, userID, token)
	return err
}
