package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/fit"
	"job-tracker-backend/internal/repository"
	appErrors "job-tracker-backend/pkg/errors"
)

// ErrNoFitProfile means there is nothing to score against: the user has not saved
// a fit profile and the request did not include one.
var ErrNoFitProfile = errors.New("no fit profile saved")

type FitService struct {
	profiles *repository.ProfileRepository
	fetcher  *fit.Fetcher
	scorer   *fit.Scorer
}

func NewFitService(profiles *repository.ProfileRepository, fetcher *fit.Fetcher, scorer *fit.Scorer) *FitService {
	return &FitService{profiles: profiles, fetcher: fetcher, scorer: scorer}
}

// Check scores one posting, fetched from its URL and/or given field by field,
// against the user's fit profile. Nothing is stored.
func (s *FitService) Check(ctx context.Context, userID string, input *domain.FitCheckInput) (*domain.FitCheckResult, error) {
	profile := strings.TrimSpace(input.FitProfile)
	if profile == "" {
		saved, err := s.profiles.Get(userID)
		if err != nil {
			return nil, err
		}
		profile = saved.FitProfile
	}
	if profile == "" {
		return nil, ErrNoFitProfile
	}

	posting := &domain.FitPosting{Source: "manual"}
	if url := strings.TrimSpace(input.URL); url != "" {
		fetched, err := s.fetcher.Fetch(ctx, url)
		if err != nil {
			return nil, err
		}
		posting = fetched
	}
	overrideIfSet(&posting.JobTitle, input.JobTitle)
	overrideIfSet(&posting.CompanyName, input.CompanyName)
	overrideIfSet(&posting.Location, input.Location)
	overrideIfSet(&posting.Description, input.Description)
	if posting.JobTitle == "" && posting.Description == "" {
		return nil, appErrors.ErrInvalidInput
	}
	posting.DescriptionLength = utf8.RuneCountInString(posting.Description)

	result, err := s.scorer.Score(ctx, profile, posting)
	if err != nil {
		return nil, err
	}
	return &domain.FitCheckResult{Fit: result, Posting: posting}, nil
}

func overrideIfSet(field *string, value string) {
	if value = strings.TrimSpace(value); value != "" {
		*field = value
	}
}
