package user

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID                       uuid.UUID  `json:"id"`
	Username                 string     `json:"username"`
	Email                    string     `json:"email"`
	PasswordHash             string     `json:"-"`
	EmailVerified            bool       `json:"email_verified"`
	EmailVerificationCode    string     `json:"-"`
	EmailVerificationExpires *time.Time `json:"-"`
	AvatarURL                *string    `json:"avatar_url"`
	Status                   string     `json:"status"`
	LastSeen                 time.Time  `json:"last_seen"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type CreateUserDTO struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type UpdateUserDTO struct {
	Username  *string `json:"username" validate:"omitempty,min=3,max=50"`
	AvatarURL *string `json:"avatar_url"`
	Status    *string `json:"status" validate:"omitempty,oneof=online offline busy away"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}
