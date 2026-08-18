package service

import (
	"regexp"
	"strings"
	"time"

	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/repository"
	appErrors "job-tracker-backend/pkg/errors"
)

type DiscoveredJobService struct {
	repo *repository.DiscoveredJobRepository
}

func NewDiscoveredJobService(repo *repository.DiscoveredJobRepository) *DiscoveredJobService {
	return &DiscoveredJobService{repo: repo}
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]`)

// normalizeCompany makes "MYOB Group Ltd" and "myob group" compare equal, so the
// applied-before flag is not defeated by punctuation or a legal suffix.
func normalizeCompany(name string) string {
	lowered := strings.ToLower(strings.TrimSpace(name))
	for _, suffix := range []string{
		" pty ltd", " pty limited", " pty", " ltd", " limited", " inc", " incorporated",
		" group", " services", " australia", " au",
	} {
		lowered = strings.TrimSuffix(lowered, suffix)
	}
	return nonAlphanumeric.ReplaceAllString(lowered, "")
}

func (s *DiscoveredJobService) retentionCutoff() time.Time {
	return time.Now().AddDate(0, 0, -domain.DiscoveredRetentionDays)
}

// GetRecent lists the retained window for the user.
func (s *DiscoveredJobService) GetRecent(userID string, includeDismissed bool) ([]domain.DiscoveredJob, error) {
	return s.repo.GetRecent(userID, includeDismissed, s.retentionCutoff())
}

func (s *DiscoveredJobService) Dismiss(userID, id string) (*domain.DiscoveredJob, error) {
	return s.repo.Dismiss(id, userID)
}

// Ingest upserts a scrape run and then enforces the retention window. Upserting on
// (user_id, external_id) is what makes a re-run idempotent, so a scraper needs no
// local de-duplication state of its own.
func (s *DiscoveredJobService) Ingest(userID string, input *domain.DiscoveredIngestInput) (*domain.DiscoveredIngestResult, error) {
	if input == nil || len(input.Jobs) == 0 {
		pruned, err := s.repo.PruneOlderThan(userID, s.retentionCutoff())
		if err != nil {
			return nil, err
		}
		return &domain.DiscoveredIngestResult{Pruned: pruned}, nil
	}

	// Companies with a real tracked application, for the applied-before flag. Done
	// here rather than in the scraper: the data already lives next to this code.
	companies, err := s.repo.CompaniesForUser(userID)
	if err != nil {
		return nil, err
	}
	applied := make(map[string]bool, len(companies))
	for _, company := range companies {
		if normalized := normalizeCompany(company); normalized != "" {
			applied[normalized] = true
		}
	}

	externalIDs := make([]string, 0, len(input.Jobs))
	for _, job := range input.Jobs {
		if job.ExternalID != "" {
			externalIDs = append(externalIDs, job.ExternalID)
		}
	}
	existing, err := s.repo.FindByExternalIDs(userID, externalIDs)
	if err != nil {
		return nil, err
	}

	result := &domain.DiscoveredIngestResult{Received: len(input.Jobs)}
	now := time.Now()

	for _, in := range input.Jobs {
		if in.ExternalID == "" || in.JobTitle == "" {
			return nil, appErrors.ErrInvalidInput
		}

		applyType := in.ApplyType
		switch applyType {
		case domain.ApplyTypeEasyApply, domain.ApplyTypeExternal, domain.ApplyTypeUnknown:
		default:
			applyType = domain.ApplyTypeUnknown
		}
		appliedBefore := applied[normalizeCompany(in.CompanyName)]

		if prev, ok := existing[in.ExternalID]; ok {
			// Keep the original DiscoveredAt so a posting stays under the day it was
			// first seen, and never resurrect a dismissed row.
			if err := s.repo.UpdateFields(prev.ID, map[string]interface{}{
				"job_title":      in.JobTitle,
				"company_name":   in.CompanyName,
				"location":       in.Location,
				"job_url":        in.JobURL,
				"source":         in.Source,
				"posted_date":    in.PostedDate,
				"apply_type":     applyType,
				"applied_before": appliedBefore,
			}); err != nil {
				return nil, err
			}
			result.Updated++
			continue
		}

		job := &domain.DiscoveredJob{
			UserID:        userID,
			ExternalID:    in.ExternalID,
			JobTitle:      in.JobTitle,
			CompanyName:   in.CompanyName,
			Location:      in.Location,
			JobURL:        in.JobURL,
			Source:        in.Source,
			PostedDate:    in.PostedDate,
			ApplyType:     applyType,
			AppliedBefore: appliedBefore,
			DiscoveredAt:  now,
		}
		if err := s.repo.Create(job); err != nil {
			return nil, err
		}
		result.Created++
	}

	pruned, err := s.repo.PruneOlderThan(userID, s.retentionCutoff())
	if err != nil {
		return nil, err
	}
	result.Pruned = pruned

	return result, nil
}
