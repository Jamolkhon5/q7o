package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"q7o/internal/email"
)

type Service struct {
	repo         *Repository
	emailService *email.Service
}

func NewService(repo *Repository, emailService *email.Service) *Service {
	return &Service{
		repo:         repo,
		emailService: emailService,
	}
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		Status:    user.Status,
		LastSeen:  user.LastSeen,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *Service) GetUserByUsername(ctx context.Context, username string) (*UserResponse, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		Status:    user.Status,
		LastSeen:  user.LastSeen,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, updates *UpdateUserDTO) (*UserResponse, error) {
	// Check if new username.go is taken
	if updates.Username != nil {
		exists, _ := s.repo.UsernameExists(ctx, *updates.Username)
		if exists {
			user, _ := s.repo.FindByID(ctx, userID)
			if user.Username != *updates.Username {
				return nil, errors.New("username already taken")
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
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			AvatarURL: user.AvatarURL,
			Status:    user.Status,
			LastSeen:  user.LastSeen,
			CreatedAt: user.CreatedAt,
		})
	}

	return responses, nil
}
