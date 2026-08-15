package service

import (
	"context"
	"errors"
	"time"

	"medcon/internal/auth"
	"medcon/internal/models"
	"medcon/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo     *repository.UserRepository
	jwtSecret    string
	jwtExpiry    int
	emailService *EmailService
	tokenStore   *PasswordResetTokenStore
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpiryHours int, emailService *EmailService, tokenStore *PasswordResetTokenStore) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		jwtSecret:    jwtSecret,
		jwtExpiry:    jwtExpiryHours,
		emailService: emailService,
		tokenStore:   tokenStore,
	}
}

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if user already exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create user
	role := models.Role(req.Role)
	user := models.NewUser(req.Name, req.Email, string(hashedPassword), role)

	// For patients, auto-verify
	if role == models.RolePatient {
		user.IsVerified = true
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, errors.New("failed to create user")
	}

	// Generate token
	token, err := auth.GenerateToken(user.ID, user.Email, string(user.Role), s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return &models.AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate token
	token, err := auth.GenerateToken(user.ID, user.Email, string(user.Role), s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return &models.AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, userID string) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("user not found")
	}

	return auth.GenerateToken(user.ID, user.Email, string(user.Role), s.jwtSecret, s.jwtExpiry)
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user != nil {
		user.PasswordHash = ""
	}
	return user, nil
}

// RequestPasswordReset sends a password reset email to the user
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		// Don't reveal if email exists or not for security
		return nil
	}

	// Generate secure token
	token, err := GenerateSecureToken(32)
	if err != nil {
		return errors.New("failed to generate reset token")
	}

	// Store token with email
	s.tokenStore.Store(email, token)

	// Send email (non-blocking in case of SMTP issues)
	go func() {
		if err := s.emailService.SendPasswordResetEmail(email, token); err != nil {
			// Log error but don't fail the request
			// In production, use a proper logger
			_ = err
		}
	}()

	return nil
}

// ResetPassword validates the token and updates the user's password
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	email, ok := s.tokenStore.Consume(token)
	if !ok {
		return errors.New("invalid or expired reset token")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	// Update user password
	updates := bson.M{
		"password_hash": string(hashedPassword),
		"updated_at":    time.Now(),
	}
	if err := s.userRepo.Update(ctx, user.ID, updates); err != nil {
		return errors.New("failed to update password")
	}

	return nil
}
