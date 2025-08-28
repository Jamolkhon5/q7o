package user

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, user *User) error {
	query := `
        INSERT INTO users (
            id, username, first_name, last_name, email, password_hash, 
            email_verification_code, email_verification_expires,
            status, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.FirstName, user.LastName,
		user.Email, user.PasswordHash,
		user.EmailVerificationCode, user.EmailVerificationExpires,
		user.Status, user.CreatedAt, user.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return err
			}
		}
	}

	return err
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
        SELECT id, username, first_name, last_name, email, password_hash, 
               email_verified, email_verification_code, email_verification_expires,
               avatar_url, status, last_seen, created_at, updated_at
        FROM users WHERE id = $1
    `

	user := &User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.FirstName, &user.LastName,
		&user.Email, &user.PasswordHash,
		&user.EmailVerified, &user.EmailVerificationCode,
		&user.EmailVerificationExpires, &user.AvatarURL,
		&user.Status, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, err
	}

	return user, err
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `
        SELECT id, username, first_name, last_name, email, password_hash, 
               email_verified, email_verification_code, email_verification_expires,
               avatar_url, status, last_seen, created_at, updated_at
        FROM users WHERE email = $1
    `

	user := &User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.FirstName, &user.LastName,
		&user.Email, &user.PasswordHash,
		&user.EmailVerified, &user.EmailVerificationCode,
		&user.EmailVerificationExpires, &user.AvatarURL,
		&user.Status, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, err
	}

	return user, err
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (*User, error) {
	query := `
        SELECT id, username, first_name, last_name, email, avatar_url, 
               status, last_seen, created_at
        FROM users WHERE username = $1
    `

	user := &User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.FirstName, &user.LastName,
		&user.Email, &user.AvatarURL,
		&user.Status, &user.LastSeen, &user.CreatedAt,
	)

	return user, err
}

func (r *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE email = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, email).Scan(&count)
	return count > 0, err
}

func (r *Repository) UsernameExists(ctx context.Context, username string) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE username = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, username).Scan(&count)
	return count > 0, err
}

func (r *Repository) FindSimilarUsernames(ctx context.Context, baseUsername string, limit int) ([]string, error) {
	query := `
        SELECT username FROM users 
        WHERE username LIKE $1 || '%'
        ORDER BY username
        LIMIT $2
    `

	rows, err := r.db.QueryContext(ctx, query, baseUsername, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		usernames = append(usernames, username)
	}

	return usernames, nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, updates *UpdateUserDTO) error {
	query := `
        UPDATE users 
        SET username = COALESCE($2, username),
            first_name = COALESCE($3, first_name),
            last_name = COALESCE($4, last_name),
            avatar_url = COALESCE($5, avatar_url),
            status = COALESCE($6, status),
            updated_at = NOW()
        WHERE id = $1
    `

	_, err := r.db.ExecContext(ctx, query, id,
		updates.Username, updates.FirstName, updates.LastName,
		updates.AvatarURL, updates.Status)
	return err
}

func (r *Repository) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_seen = NOW(), status = 'online' WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status)
	return err
}

func (r *Repository) VerifyEmail(ctx context.Context, id uuid.UUID) error {
	query := `
        UPDATE users 
        SET email_verified = true,
            email_verification_code = NULL,
            email_verification_expires = NULL,
            updated_at = NOW()
        WHERE id = $1
    `
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) UpdateVerificationCode(ctx context.Context, id uuid.UUID, code string, expires time.Time) error {
	query := `
        UPDATE users 
        SET email_verification_code = $2,
            email_verification_expires = $3,
            updated_at = NOW()
        WHERE id = $1
    `
	_, err := r.db.ExecContext(ctx, query, id, code, expires)
	return err
}

func (r *Repository) Search(ctx context.Context, query string, limit, offset int) ([]*User, error) {
	sqlQuery := `
        SELECT id, username, first_name, last_name, email, avatar_url, 
               status, last_seen, created_at
        FROM users 
        WHERE username ILIKE $1 OR email ILIKE $1 
              OR first_name ILIKE $1 OR last_name ILIKE $1
              OR CONCAT(first_name, ' ', last_name) ILIKE $1
        ORDER BY username
        LIMIT $2 OFFSET $3
    `

	searchPattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, sqlQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.FirstName, &user.LastName,
			&user.Email, &user.AvatarURL,
			&user.Status, &user.LastSeen, &user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
