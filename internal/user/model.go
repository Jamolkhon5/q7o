package user

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID                       uuid.UUID  `json:"id"`
	Username                 string     `json:"username"` // ✅ ИСПРАВЛЕНО
	FirstName                string     `json:"first_name"`
	LastName                 string     `json:"last_name"`
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
	// Extended profile fields
	Phone       *string    `json:"phone"`
	Bio         *string    `json:"bio"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Location    *string    `json:"location"`
	Timezone    *string    `json:"timezone"`
}

type CreateUserDTO struct {
	FirstName string `json:"first_name" validate:"required,min=2,max=100"`
	LastName  string `json:"last_name" validate:"required,min=2,max=100"`
	Username  string `json:"username" validate:"omitempty,min=3,max=50"` // ✅ ИСПРАВЛЕНО
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
}

type UpdateUserDTO struct {
	Username  *string `json:"username" validate:"omitempty,min=3,max=50"` // ✅ ИСПРАВЛЕНО
	FirstName *string `json:"first_name" validate:"omitempty,min=2,max=100"`
	LastName  *string `json:"last_name" validate:"omitempty,min=2,max=100"`
	AvatarURL *string `json:"avatar_url"`
	Status    *string `json:"status" validate:"omitempty,oneof=online offline busy away"`
	// Extended profile fields
	Phone       *string    `json:"phone" validate:"omitempty,e164"`
	Bio         *string    `json:"bio" validate:"omitempty,max=500"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Location    *string    `json:"location" validate:"omitempty,max=100"`
	Timezone    *string    `json:"timezone" validate:"omitempty"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"` // ✅ ИСПРАВЛЕНО
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	// Extended profile fields
	Phone       *string    `json:"phone"`
	Bio         *string    `json:"bio"`
	DateOfBirth *time.Time `json:"date_of_birth"`
	Location    *string    `json:"location"`
	Timezone    *string    `json:"timezone"`
}

type CheckUsernameResponse struct {
	Available   bool     `json:"available"`
	Username    string   `json:"username"` // ✅ ИСПРАВЛЕНО
	Suggestions []string `json:"suggestions,omitempty"`
}

type ChangePasswordDTO struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}
