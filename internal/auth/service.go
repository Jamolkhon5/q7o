package auth

import (
	"context"
	"errors"
	"time"

	"q7o/config"
	"q7o/internal/common/utils"
	"q7o/internal/email"
	"q7o/internal/user"

	"github.com/google/uuid"
)

type Service struct {
	repo         *Repository
	userRepo     *user.Repository
	emailService *email.Service
	jwtConfig    config.JWTConfig
}

func NewService(repo *Repository, userRepo *user.Repository, emailService *email.Service, jwtConfig config.JWTConfig) *Service {
	return &Service{
		repo:         repo,
		userRepo:     userRepo,
		emailService: emailService,
		jwtConfig:    jwtConfig,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*user.UserResponse, *TokenPair, error) {
	// Check if email exists
	if exists, _ := s.userRepo.EmailExists(ctx, req.Email); exists {
		return nil, nil, errors.New("email already exists")
	}

	// Check if username exists
	if exists, _ := s.userRepo.UsernameExists(ctx, req.Username); exists {
		return nil, nil, errors.New("username already exists")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, nil, err
	}

	// Generate verification code
	verificationCode := utils.GenerateCode(6)
	expiresAt := time.Now().Add(15 * time.Minute)

	// Create user
	newUser := &user.User{
		ID:                       uuid.New(),
		Username:                 req.Username,
		Email:                    req.Email,
		PasswordHash:             hashedPassword,
		EmailVerified:            false,
		EmailVerificationCode:    verificationCode,
		EmailVerificationExpires: &expiresAt,
		Status:                   "offline",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, nil, err
	}

	// Send verification email
	go s.emailService.SendVerificationEmail(req.Email, req.Username, verificationCode)

	// Generate tokens
	tokens, err := GenerateTokenPair(newUser.ID, newUser.Username, s.jwtConfig)
	if err != nil {
		return nil, nil, err
	}

	// Save refresh token
	if err := s.repo.SaveRefreshToken(ctx, newUser.ID, tokens.RefreshToken, ""); err != nil {
		return nil, nil, err
	}

	userResponse := &user.UserResponse{
		ID:        newUser.ID,
		Username:  newUser.Username,
		Email:     newUser.Email,
		Status:    newUser.Status,
		CreatedAt: newUser.CreatedAt,
	}

	return userResponse, tokens, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*user.UserResponse, *TokenPair, error) {
	// Find user by email
	u, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Verify password
	if !utils.CheckPassword(req.Password, u.PasswordHash) {
		return nil, nil, errors.New("invalid credentials")
	}

	// Generate tokens
	tokens, err := GenerateTokenPair(u.ID, u.Username, s.jwtConfig)
	if err != nil {
		return nil, nil, err
	}

	// Save refresh token
	if err := s.repo.SaveRefreshToken(ctx, u.ID, tokens.RefreshToken, ""); err != nil {
		return nil, nil, err
	}

	// Update last seen
	s.userRepo.UpdateLastSeen(ctx, u.ID)

	userResponse := &user.UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}

	return userResponse, tokens, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := ValidateToken(refreshToken, s.jwtConfig.RefreshSecret)
	if err != nil {
		return nil, err
	}

	// Check if refresh token exists in database
	exists, err := s.repo.RefreshTokenExists(ctx, claims.UserID, refreshToken)
	if err != nil || !exists {
		return nil, errors.New("invalid refresh token")
	}

	// Generate new token pair
	tokens, err := GenerateTokenPair(claims.UserID, claims.Username, s.jwtConfig)
	if err != nil {
		return nil, err
	}

	// Update refresh token
	if err := s.repo.UpdateRefreshToken(ctx, claims.UserID, refreshToken, tokens.RefreshToken); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *Service) VerifyEmail(ctx context.Context, email, code string) error {
	u, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	if u.EmailVerified {
		return errors.New("email already verified")
	}

	if u.EmailVerificationCode != code {
		return errors.New("invalid verification code")
	}

	if u.EmailVerificationExpires != nil && time.Now().After(*u.EmailVerificationExpires) {
		return errors.New("verification code expired")
	}

	return s.userRepo.VerifyEmail(ctx, u.ID)
}

func (s *Service) ResendVerification(ctx context.Context, email string) error {
	u, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	if u.EmailVerified {
		return errors.New("email already verified")
	}

	// Generate new code
	verificationCode := utils.GenerateCode(6)
	expiresAt := time.Now().Add(15 * time.Minute)

	// Update verification code
	if err := s.userRepo.UpdateVerificationCode(ctx, u.ID, verificationCode, expiresAt); err != nil {
		return err
	}

	// Send email
	go s.emailService.SendVerificationEmail(email, u.Username, verificationCode)

	return nil
}

func (s *Service) Logout(ctx context.Context, userID string, refreshToken string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	return s.repo.DeleteRefreshToken(ctx, uid, refreshToken)
}

func (s *Service) ValidateAndGetUser(ctx context.Context, userID string) (*user.UserResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	u, err := s.userRepo.FindByID(ctx, uid)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Обновляем last seen
	s.userRepo.UpdateLastSeen(ctx, uid)

	return &user.UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}, nil
}
