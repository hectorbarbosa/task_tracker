package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"task_tracker/internal/middleware"
	"task_tracker/internal/model"
	"task_tracker/internal/repository"
)

// AuthService handles user registration and authentication.
type AuthService struct {
	userRepo          *repository.UserRepository
	jwtSecret         string
	jwtExpirationHours int
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpirationHours int) *AuthService {
	return &AuthService{
		userRepo:          userRepo,
		jwtSecret:         jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

// Register creates a new user account and returns a JWT token.
func (s *AuthService) Register(ctx context.Context, input model.RegisterInput) (*model.TokenResponse, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &model.User{
		Email:        input.Email,
		PasswordHash: string(passwordHash),
		Name:         input.Name,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate JWT token
	token, expiresAt, err := middleware.GenerateToken(user.ID, s.jwtSecret, s.jwtExpirationHours)
	if err != nil {
		return nil, err
	}

	return &model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// Login authenticates a user and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, input model.LoginInput) (*model.TokenResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate JWT token
	token, expiresAt, err := middleware.GenerateToken(user.ID, s.jwtSecret, s.jwtExpirationHours)
	if err != nil {
		return nil, err
	}

	return &model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}
