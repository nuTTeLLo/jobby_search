package service

import (
	"strings"
	"time"
	"unicode/utf8"

	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/repository"
	appErrors "job-tracker-backend/pkg/errors"
)

type ProfileService struct {
	repo *repository.ProfileRepository
}

func NewProfileService(repo *repository.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) Get(userID string) (*domain.Profile, error) {
	return s.repo.Get(userID)
}

func (s *ProfileService) Update(userID string, input *domain.ProfileInput) (*domain.Profile, error) {
	fitProfile := strings.TrimSpace(input.FitProfile)
	// Characters, not bytes, to agree with the counter on the Profile page.
	if utf8.RuneCountInString(fitProfile) > domain.MaxFitProfileLength {
		return nil, appErrors.ErrInvalidInput
	}

	now := time.Now()
	profile := &domain.Profile{UserID: userID, FitProfile: fitProfile, UpdatedAt: &now}
	if err := s.repo.Save(profile); err != nil {
		return nil, err
	}
	return profile, nil
}
