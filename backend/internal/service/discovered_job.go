package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/repository"
	appErrors "job-tracker-backend/pkg/errors"
)

type DiscoveredJobService struct {
	repo     *repository.DiscoveredJobRepository
	profiles *repository.ProfileRepository
}

func NewDiscoveredJobService(repo *repository.DiscoveredJobRepository, profiles *repository.ProfileRepository) *DiscoveredJobService {
	return &DiscoveredJobService{repo: repo, profiles: profiles}
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]`)

// companyAliases maps a normalized company name to the one it is known by elsewhere,
// for brand variants no suffix rule can catch. Keys and values are normalized names.
var companyAliases = map[string]string{
	// Seek lists Allume as "Allume Energy", LinkedIn as "Allume ANZ".
	"allumeenergy": "allume",
}

// normalizeCompany makes "MYOB Group Ltd" and "myob group" compare equal, so the
// applied-before flag is not defeated by punctuation or a legal suffix.
func normalizeCompany(name string) string {
	lowered := strings.ToLower(strings.TrimSpace(name))
	for _, suffix := range []string{
		" pty ltd", " pty limited", " pty", " ltd", " limited", " inc", " incorporated",
		" group", " services", " of australia", " australia", " au",
		" australia and new zealand", " australia & new zealand", " new zealand", " anz", " nz",
	} {
		lowered = strings.TrimSuffix(lowered, suffix)
	}
	normalized := nonAlphanumeric.ReplaceAllString(lowered, "")
	if alias, ok := companyAliases[normalized]; ok {
		return alias
	}
	return normalized
}

// matchKey groups the same role scraped from different boards. The boards only
// expose the employer's apply link publicly for Indeed, so the key is company plus
// title: "Senior Full-Stack Engineer" at "Acme Pty Ltd" on Seek and "Senior Full
// Stack Engineer" at "Acme" on LinkedIn share one. A board that words the company
// differently ("CBA" vs "Commonwealth Bank") is not matched; this is best effort.
func matchKey(job *domain.DiscoveredJob) string {
	company := normalizeCompany(job.CompanyName)
	title := nonAlphanumeric.ReplaceAllString(strings.ToLower(job.JobTitle), "")
	if company == "" || title == "" {
		// Nothing reliable to match on; keep the posting on its own.
		return "id:" + job.ID
	}
	return company + "|" + title
}

// fitAssessment is a validated fit from the scraper, ready to store.
type fitAssessment struct {
	score     int
	summary   string
	strengths domain.StringList
	gaps      domain.StringList
	flags     domain.StringList
	model     string
	scoredAt  time.Time
}

// newFitAssessment validates a scraper's fit, returning nil when there is nothing
// to store. An out-of-range score is dropped rather than failing the ingest: the
// posting itself is still worth keeping.
func newFitAssessment(in *domain.Fit, scoredAt time.Time) *fitAssessment {
	if in == nil || in.Score < domain.MinFitScore || in.Score > domain.MaxFitScore {
		return nil
	}
	return &fitAssessment{
		score:     in.Score,
		summary:   strings.TrimSpace(in.Summary),
		strengths: cleanList(in.Strengths),
		gaps:      cleanList(in.Gaps),
		flags:     cleanList(in.Flags),
		model:     in.Model,
		scoredAt:  scoredAt,
	}
}

func (f *fitAssessment) applyTo(job *domain.DiscoveredJob) {
	score, scoredAt := f.score, f.scoredAt
	job.FitScore = &score
	job.FitSummary = f.summary
	job.FitStrengths = f.strengths
	job.FitGaps = f.gaps
	job.FitFlags = f.flags
	job.FitModel = f.model
	job.FitScoredAt = &scoredAt
}

func (f *fitAssessment) addColumns(fields map[string]interface{}) {
	fields["fit_score"] = f.score
	fields["fit_summary"] = f.summary
	fields["fit_strengths"] = f.strengths
	fields["fit_gaps"] = f.gaps
	fields["fit_flags"] = f.flags
	fields["fit_model"] = f.model
	fields["fit_scored_at"] = f.scoredAt
}

func cleanList(items []string) domain.StringList {
	cleaned := domain.StringList{}
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return cleaned
}

func (s *DiscoveredJobService) retentionCutoff() time.Time {
	return time.Now().AddDate(0, 0, -domain.DiscoveredRetentionDays)
}

// GetRecent lists the retained window for the user.
func (s *DiscoveredJobService) GetRecent(userID string, includeDismissed bool) ([]domain.DiscoveredJob, error) {
	jobs, err := s.repo.GetRecent(userID, includeDismissed, s.retentionCutoff())
	if err != nil {
		return nil, err
	}
	applied, err := s.repo.LastAppliedByCompany(userID)
	if err != nil {
		return nil, fmt.Errorf("load last applied dates: %w", err)
	}
	// Several spellings of a company can normalize to one; keep the latest date.
	lastApplied := make(map[string]time.Time, len(applied))
	for _, row := range applied {
		key := normalizeCompany(row.CompanyName)
		if key != "" && row.LastAppliedAt.After(lastApplied[key]) {
			lastApplied[key] = row.LastAppliedAt
		}
	}

	for i := range jobs {
		jobs[i].MatchKey = matchKey(&jobs[i])
		if at, ok := lastApplied[normalizeCompany(jobs[i].CompanyName)]; ok {
			jobs[i].LastAppliedAt = &at
		}
	}
	return jobs, nil
}

// Unscored lists the postings the fit sweep should score: never scored, or scored
// before the fit profile was last saved.
func (s *DiscoveredJobService) Unscored(userID string) ([]domain.UnscoredDiscoveredJob, error) {
	profile, err := s.profiles.Get(userID)
	if err != nil {
		return nil, fmt.Errorf("load profile: %w", err)
	}
	jobs, err := s.repo.FindUnscored(userID, s.retentionCutoff(), profile.UpdatedAt)
	if err != nil {
		return nil, err
	}

	unscored := make([]domain.UnscoredDiscoveredJob, 0, len(jobs))
	for i := range jobs {
		job := &jobs[i]
		unscored = append(unscored, domain.UnscoredDiscoveredJob{
			ID:          job.ID,
			MatchKey:    matchKey(job),
			JobTitle:    job.JobTitle,
			CompanyName: job.CompanyName,
			Location:    job.Location,
			Source:      job.Source,
			JobURL:      job.JobURL,
			Description: job.Description,
		})
	}
	return unscored, nil
}

// SetFit stores a fit for one posting, as scored by the fit sweep.
func (s *DiscoveredJobService) SetFit(userID, id string, in *domain.Fit) error {
	fit := newFitAssessment(in, time.Now())
	if fit == nil {
		return appErrors.ErrInvalidInput
	}
	fields := map[string]interface{}{}
	fit.addColumns(fields)
	return s.repo.UpdateFieldsForUser(id, userID, fields)
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

		fit := newFitAssessment(in.Fit, now)

		if prev, ok := existing[in.ExternalID]; ok {
			// Keep the original DiscoveredAt so a posting stays under the day it was
			// first seen, and never resurrect a dismissed row.
			fields := map[string]interface{}{
				"job_title":      in.JobTitle,
				"company_name":   in.CompanyName,
				"location":       in.Location,
				"job_url":        in.JobURL,
				"apply_url":      in.ApplyURL,
				"source":         in.Source,
				"posted_date":    in.PostedDate,
				"apply_type":     applyType,
				"applied_before": appliedBefore,
			}
			// An unscored re-scrape keeps the existing assessment, and a missing
			// description keeps the stored one.
			if fit != nil {
				fit.addColumns(fields)
			}
			if in.Description != "" {
				fields["description"] = in.Description
			}
			if err := s.repo.UpdateFields(prev.ID, fields); err != nil {
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
			ApplyURL:      in.ApplyURL,
			Source:        in.Source,
			PostedDate:    in.PostedDate,
			ApplyType:     applyType,
			AppliedBefore: appliedBefore,
			Description:   in.Description,
			DiscoveredAt:  now,
		}
		if fit != nil {
			fit.applyTo(job)
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
