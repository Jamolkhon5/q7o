package user

import (
	"context"
	"errors"
	"mime/multipart"
	"regexp"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"q7o/internal/email"
	"q7o/internal/upload"
)

type Service struct {
	repo          *Repository
	emailService  *email.Service
	uploadService *upload.Service
}

func NewService(repo *Repository, emailService *email.Service, uploadService *upload.Service) *Service {
	return &Service{
		repo:          repo,
		emailService:  emailService,
		uploadService: uploadService,
	}
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		AvatarURL:   user.AvatarURL,
		Status:      user.Status,
		LastSeen:    user.LastSeen,
		CreatedAt:   user.CreatedAt,
		Phone:       user.Phone,
		Bio:         user.Bio,
		DateOfBirth: user.DateOfBirth,
		Location:    user.Location,
		Timezone:    user.Timezone,
	}, nil
}

func (s *Service) GetUserByUsername(ctx context.Context, username string) (*UserResponse, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		AvatarURL:   user.AvatarURL,
		Status:      user.Status,
		LastSeen:    user.LastSeen,
		CreatedAt:   user.CreatedAt,
		Phone:       user.Phone,
		Bio:         user.Bio,
		DateOfBirth: user.DateOfBirth,
		Location:    user.Location,
		Timezone:    user.Timezone,
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, updates *UpdateUserDTO) (*UserResponse, error) {
	// Check if new username is taken
	if updates.Username != nil {
		exists, _ := s.repo.UsernameExists(ctx, *updates.Username)
		if exists {
			user, _ := s.repo.FindByID(ctx, userID)
			if user.Username != *updates.Username {
				return nil, errors.New("username already taken")
			}
		}
	}

	// Check if new phone number is taken
	if updates.Phone != nil && *updates.Phone != "" {
		exists, _ := s.repo.PhoneExists(ctx, *updates.Phone)
		if exists {
			user, _ := s.repo.FindByID(ctx, userID)
			if user.Phone == nil || *user.Phone != *updates.Phone {
				return nil, errors.New("phone number already taken")
			}
		}
	}

	if err := s.repo.Update(ctx, userID, updates); err != nil {
		return nil, err
	}

	return s.GetUserByID(ctx, userID)
}

func (s *Service) UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error {
	return s.repo.UpdateStatus(ctx, userID, status)
}

func (s *Service) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*UserResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	users, err := s.repo.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	var responses []*UserResponse
	for _, user := range users {
		responses = append(responses, &UserResponse{
			ID:          user.ID,
			Username:    user.Username,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			Email:       user.Email,
			AvatarURL:   user.AvatarURL,
			Status:      user.Status,
			LastSeen:    user.LastSeen,
			CreatedAt:   user.CreatedAt,
			Phone:       user.Phone,
			Bio:         user.Bio,
			DateOfBirth: user.DateOfBirth,
			Location:    user.Location,
			Timezone:    user.Timezone,
		})
	}

	return responses, nil
}

func (s *Service) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, avatarURL string) (*UserResponse, error) {
	if err := s.repo.UpdateAvatarURL(ctx, userID, avatarURL); err != nil {
		return nil, err
	}

	return s.GetUserByID(ctx, userID)
}

func (s *Service) UploadAvatar(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (*UserResponse, error) {
	// Get current user to check for existing avatar
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Delete existing avatar if it exists
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		if err := s.uploadService.DeleteAvatar(*user.AvatarURL); err != nil {
			// Log error but don't fail the upload
		}
	}

	// Upload new avatar
	result, err := s.uploadService.UploadAvatar(file)
	if err != nil {
		return nil, err
	}

	// Update user's avatar URL in database
	if err := s.repo.UpdateAvatarURL(ctx, userID, result.URL); err != nil {
		// Clean up uploaded file on database error
		s.uploadService.DeleteAvatar(result.Filename)
		return nil, err
	}

	return s.GetUserByID(ctx, userID)
}

func (s *Service) DeleteAvatar(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	// Get current user to check for existing avatar
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Delete avatar file if it exists
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		if err := s.uploadService.DeleteAvatar(*user.AvatarURL); err != nil {
			// Log error but don't fail the operation
		}
	}

	// Update database to remove avatar URL
	if err := s.repo.UpdateAvatarURL(ctx, userID, ""); err != nil {
		return nil, err
	}

	return s.GetUserByID(ctx, userID)
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, dto *ChangePasswordDTO) error {
	// Get current user
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(dto.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password in database
	return s.repo.UpdatePassword(ctx, userID, string(newPasswordHash))
}

func (s *Service) ValidatePhoneNumber(phone string) error {
	if phone == "" {
		return nil // Empty phone is valid
	}

	// E.164 format validation (simplified)
	phoneRegex := regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(phone) {
		return errors.New("phone number must be in international format (e.g., +1234567890)")
	}

	return nil
}
