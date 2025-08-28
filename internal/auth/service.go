package auth

import (
	"context"
	"errors"
	"strings"
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

	// Определяем username.go
	var finalUsername string

	if req.Username != "" {
		// Если пользователь предоставил username.go, проверяем его
		req.Username = strings.ToLower(strings.TrimSpace(req.Username))
		if exists, _ := s.userRepo.UsernameExists(ctx, req.Username); exists {
			return nil, nil, errors.New("username.go already exists")
		}
		finalUsername = req.Username
	} else {
		// Генерируем username.go автоматически
		suggestions := utils.GenerateUsername(req.FirstName, req.LastName)

		// Находим первый свободный вариант
		for _, suggestion := range suggestions {
			if exists, _ := s.userRepo.UsernameExists(ctx, suggestion); !exists {
				finalUsername = suggestion
				break
			}
		}

		// Если все варианты заняты, генерируем с суффиксом
		if finalUsername == "" {
			if len(suggestions) > 0 {
				// Получаем все похожие usernames для проверки
				baseUsername := suggestions[0]
				similarUsernames, _ := s.userRepo.FindSimilarUsernames(ctx, baseUsername, 100)
				finalUsername = utils.GenerateUsernameWithSuffix(baseUsername, similarUsernames)
			} else {
				// Крайний случай - используем email prefix
				emailPrefix := strings.Split(req.Email, "@")[0]
				emailPrefix = utils.CleanString(emailPrefix)
				if len(emailPrefix) > 20 {
					emailPrefix = emailPrefix[:20]
				}
				similarUsernames, _ := s.userRepo.FindSimilarUsernames(ctx, emailPrefix, 100)
				finalUsername = utils.GenerateUsernameWithSuffix(emailPrefix, similarUsernames)
			}
		}
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
		Username:                 finalUsername,
		FirstName:                req.FirstName,
		LastName:                 req.LastName,
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

	// Send verification email with full name
	fullName := req.FirstName + " " + req.LastName
	go s.emailService.SendVerificationEmail(req.Email, fullName, verificationCode)

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
		FirstName: newUser.FirstName,
		LastName:  newUser.LastName,
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
		FirstName: u.FirstName,
		LastName:  u.LastName,
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

	// Send email with full name
	fullName := u.FirstName + " " + u.LastName
	go s.emailService.SendVerificationEmail(email, fullName, verificationCode)

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
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}, nil
}

func (s *Service) CheckUsernameAvailability(ctx context.Context, username string, firstName string, lastName string) (*user.CheckUsernameResponse, error) {
	// Приводим к нижнему регистру и очищаем
	username = strings.ToLower(strings.TrimSpace(username))

	// Проверяем доступность
	exists, err := s.userRepo.UsernameExists(ctx, username)
	if err != nil {
		return nil, err
	}

	response := &user.CheckUsernameResponse{
		Available: !exists,
		Username:  username,
	}

	// Если username.go занят, генерируем альтернативы
	if exists && firstName != "" && lastName != "" {
		alternatives := utils.GenerateAlternatives(username, firstName, lastName, 5)

		// Проверяем доступность каждой альтернативы
		availableAlternatives := []string{}
		for _, alt := range alternatives {
			if altExists, _ := s.userRepo.UsernameExists(ctx, alt); !altExists {
				availableAlternatives = append(availableAlternatives, alt)
			}
		}

		// Если все предложенные альтернативы заняты, генерируем с суффиксом
		if len(availableAlternatives) == 0 {
			similarUsernames, _ := s.userRepo.FindSimilarUsernames(ctx, username, 100)
			newSuggestion := utils.GenerateUsernameWithSuffix(username, similarUsernames)
			availableAlternatives = append(availableAlternatives, newSuggestion)
		}

		response.Suggestions = availableAlternatives
	}

	return response, nil
}

func (s *Service) GenerateUsernameSuggestions(ctx context.Context, firstName string, lastName string) ([]string, error) {
	// Генерируем базовые варианты
	variants := utils.GenerateUsername(firstName, lastName)

	// Фильтруем только доступные
	availableSuggestions := []string{}
	for _, variant := range variants {
		if exists, _ := s.userRepo.UsernameExists(ctx, variant); !exists {
			availableSuggestions = append(availableSuggestions, variant)
			if len(availableSuggestions) >= 5 {
				break
			}
		}
	}

	// Если мало доступных, генерируем с суффиксами
	if len(availableSuggestions) < 3 && len(variants) > 0 {
		baseUsername := variants[0]
		similarUsernames, _ := s.userRepo.FindSimilarUsernames(ctx, baseUsername, 100)

		for i := 1; len(availableSuggestions) < 5 && i < 100; i++ {
			candidate := utils.GenerateUsernameWithSuffix(baseUsername, similarUsernames)
			if exists, _ := s.userRepo.UsernameExists(ctx, candidate); !exists {
				availableSuggestions = append(availableSuggestions, candidate)
				similarUsernames = append(similarUsernames, candidate)
			}
		}
	}

	return availableSuggestions, nil
}
