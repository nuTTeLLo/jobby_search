package service

import (
	"strings"
	"time"

	"job-tracker-backend/internal/auth"
	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/repository"
	appErrors "job-tracker-backend/pkg/errors"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

func NewAuthService(repo *repository.UserRepository, secret string, expiry time.Duration) *AuthService {
	return &AuthService{userRepo: repo, jwtSecret: secret, jwtExpiry: expiry}
}

func (s *AuthService) Register(input *domain.RegisterInput) (*domain.AuthResponse, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Email == "" || input.Password == "" {
		return nil, appErrors.ErrInvalidInput
	}
	if len(input.Password) < 8 {
		return nil, appErrors.ErrInvalidInput
	}

	_, err := s.userRepo.GetByEmail(input.Email)
	if err == nil {
		return nil, appErrors.ErrAlreadyExists
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{Email: input.Email, PasswordHash: hash}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return s.buildAuthResponse(user)
}

func (s *AuthService) Login(input *domain.LoginInput) (*domain.AuthResponse, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.userRepo.GetByEmail(input.Email)
	if err != nil {
		return nil, appErrors.ErrInvalidInput // don't leak "user not found"
	}

	if !auth.CheckPasswordHash(input.Password, user.PasswordHash) {
		return nil, appErrors.ErrInvalidInput
	}

	return s.buildAuthResponse(user)
}

func (s *AuthService) ChangePassword(userID string, input *domain.ChangePasswordInput) error {
	if len(input.NewPassword) < 8 {
		return appErrors.ErrInvalidInput
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	if !auth.CheckPasswordHash(input.CurrentPassword, user.PasswordHash) {
		return appErrors.ErrInvalidInput
	}

	hash, err := auth.HashPassword(input.NewPassword)
	if err != nil {
		return err
	}

	return s.userRepo.UpdatePasswordHash(userID, hash)
}

func (s *AuthService) buildAuthResponse(user *domain.User) (*domain.AuthResponse, error) {
	token, err := auth.GenerateToken(user.ID, user.Email, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}
	return &domain.AuthResponse{
		Token: token,
		User:  domain.AuthUser{ID: user.ID, Email: user.Email},
	}, nil
}
