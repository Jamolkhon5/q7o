package call

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Call struct {
	ID           uuid.UUID  `json:"id"`
	RoomName     string     `json:"room_name"`
	CallerID     uuid.UUID  `json:"caller_id"`
	CalleeID     uuid.UUID  `json:"callee_id"`
	CallerName   string     `json:"caller_name"` // Добавить
	CalleeName   string     `json:"callee_name"` // Добавить
	CallType     string     `json:"call_type"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	AnsweredAt   *time.Time `json:"answered_at,omitempty"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	Duration     int        `json:"duration"`
	RecordingURL *string    `json:"recording_url,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, call *Call) error {
	query := `
        INSERT INTO calls (
            id, room_name, caller_id, callee_id, call_type,
            status, started_at, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	_, err := r.db.ExecContext(ctx, query,
		call.ID, call.RoomName, call.CallerID, call.CalleeID,
		call.CallType, call.Status, call.StartedAt, time.Now(),
	)

	return err
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Call, error) {
	query := `
        SELECT id, room_name, caller_id, callee_id, call_type,
               status, started_at, answered_at, ended_at, duration,
               recording_url, created_at
        FROM calls WHERE id = $1
    `

	call := &Call{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&call.ID, &call.RoomName, &call.CallerID, &call.CalleeID,
		&call.CallType, &call.Status, &call.StartedAt,
		&call.AnsweredAt, &call.EndedAt, &call.Duration,
		&call.RecordingURL, &call.CreatedAt,
	)

	return call, err
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, answeredAt, endedAt *time.Time) error {
	query := `
        UPDATE calls 
        SET status = $2, answered_at = $3, ended_at = $4
        WHERE id = $1
    `

	_, err := r.db.ExecContext(ctx, query, id, status, answeredAt, endedAt)
	return err
}

func (r *Repository) UpdateDuration(ctx context.Context, id uuid.UUID, duration int) error {
	query := `UPDATE calls SET duration = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, duration)
	return err
}

func (r *Repository) GetUserCalls(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Call, error) {
	query := `
        SELECT id, room_name, caller_id, callee_id, call_type,
               status, started_at, answered_at, ended_at, duration,
               recording_url, created_at
        FROM calls 
        WHERE caller_id = $1 OR callee_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []*Call
	for rows.Next() {
		call := &Call{}
		err := rows.Scan(
			&call.ID, &call.RoomName, &call.CallerID, &call.CalleeID,
			&call.CallType, &call.Status, &call.StartedAt,
			&call.AnsweredAt, &call.EndedAt, &call.Duration,
			&call.RecordingURL, &call.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		calls = append(calls, call)
	}

	return calls, nil
}
