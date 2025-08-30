package settings

import (
	"fmt"
	"github.com/google/uuid"
)

type Service interface {
	GetUserSettings(userID uuid.UUID) (*SettingsResponse, error)
	UpdateUserSettings(userID uuid.UUID, dto *UpdateSettingsDTO) (*SettingsResponse, error)
	DeleteUserSettings(userID uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetUserSettings(userID uuid.UUID) (*SettingsResponse, error) {
	settings, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	return settings.ToResponse(), nil
}

func (s *service) UpdateUserSettings(userID uuid.UUID, dto *UpdateSettingsDTO) (*SettingsResponse, error) {
	settings, err := s.repo.Update(userID, dto)
	if err != nil {
		return nil, fmt.Errorf("failed to update user settings: %w", err)
	}

	return settings.ToResponse(), nil
}

func (s *service) DeleteUserSettings(userID uuid.UUID) error {
	err := s.repo.Delete(userID)
	if err != nil {
		return fmt.Errorf("failed to delete user settings: %w", err)
	}

	return nil
}