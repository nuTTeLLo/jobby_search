package repository

import (
	"time"

	"job-tracker-backend/internal/domain"
	appErrors "job-tracker-backend/pkg/errors"

	"gorm.io/gorm"
)

type DiscoveredJobRepository struct {
	db *gorm.DB
}

func NewDiscoveredJobRepository(db *gorm.DB) *DiscoveredJobRepository {
	return &DiscoveredJobRepository{db: db}
}

// GetRecent returns the retained window, newest first. Dismissed rows are hidden
// unless asked for.
func (r *DiscoveredJobRepository) GetRecent(userID string, includeDismissed bool, since time.Time) ([]domain.DiscoveredJob, error) {
	var jobs []domain.DiscoveredJob
	query := r.db.Model(&domain.DiscoveredJob{}).
		Where("user_id = ? AND discovered_at >= ?", userID, since)

	if !includeDismissed {
		query = query.Where("dismissed = ?", false)
	}

	if err := query.Order("discovered_at DESC, company_name ASC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// FindUnscored returns the retained, undismissed postings that have no fit, or
// whose fit predates profileSavedAt (nil when there is no saved profile).
func (r *DiscoveredJobRepository) FindUnscored(userID string, since time.Time, profileSavedAt *time.Time) ([]domain.DiscoveredJob, error) {
	query := r.db.Where("user_id = ? AND discovered_at >= ? AND dismissed = ?", userID, since, false)
	if profileSavedAt != nil {
		query = query.Where("fit_score IS NULL OR fit_scored_at < ?", *profileSavedAt)
	} else {
		query = query.Where("fit_score IS NULL")
	}

	var jobs []domain.DiscoveredJob
	if err := query.Order("discovered_at DESC, company_name ASC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// UpdateFieldsForUser is UpdateFields scoped to one user's rows, for writes that
// come straight from a request.
func (r *DiscoveredJobRepository) UpdateFieldsForUser(id, userID string, fields map[string]interface{}) error {
	result := r.db.Model(&domain.DiscoveredJob{}).Where("id = ? AND user_id = ?", id, userID).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return appErrors.ErrNotFound
	}
	return nil
}

// FindByExternalIDs loads existing rows for this user keyed by external id, so an
// ingest can tell a create from an update without a query per posting.
func (r *DiscoveredJobRepository) FindByExternalIDs(userID string, externalIDs []string) (map[string]domain.DiscoveredJob, error) {
	existing := make(map[string]domain.DiscoveredJob)
	if len(externalIDs) == 0 {
		return existing, nil
	}

	var jobs []domain.DiscoveredJob
	if err := r.db.Where("user_id = ? AND external_id IN ?", userID, externalIDs).Find(&jobs).Error; err != nil {
		return nil, err
	}
	for _, job := range jobs {
		existing[job.ExternalID] = job
	}
	return existing, nil
}

func (r *DiscoveredJobRepository) Create(job *domain.DiscoveredJob) error {
	return r.db.Create(job).Error
}

// UpdateFields refreshes the mutable columns of an existing row. Dismissed is
// deliberately not touched: re-scraping a posting must not un-dismiss it.
func (r *DiscoveredJobRepository) UpdateFields(id string, fields map[string]interface{}) error {
	return r.db.Model(&domain.DiscoveredJob{}).Where("id = ?", id).Updates(fields).Error
}

func (r *DiscoveredJobRepository) Dismiss(id, userID string) (*domain.DiscoveredJob, error) {
	result := r.db.Model(&domain.DiscoveredJob{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("dismissed", true)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, appErrors.ErrNotFound
	}

	var job domain.DiscoveredJob
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// PruneOlderThan enforces the rolling retention window and reports how many rows
// it removed.
func (r *DiscoveredJobRepository) PruneOlderThan(userID string, cutoff time.Time) (int, error) {
	result := r.db.Where("user_id = ? AND discovered_at < ?", userID, cutoff).
		Delete(&domain.DiscoveredJob{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// CompaniesForUser returns every company the user has an actual tracked job for.
// Used to flag discovered postings from places already applied to.
func (r *DiscoveredJobRepository) CompaniesForUser(userID string) ([]string, error) {
	var companies []string
	if err := r.db.Model(&domain.Job{}).
		Where("user_id = ? AND company_name <> ''", userID).
		Distinct().
		Pluck("company_name", &companies).Error; err != nil {
		return nil, err
	}
	return companies, nil
}

// AppliedJob is one application the user actually sent, as named on their tracked
// jobs. RejectedAt is nil unless the application currently stands rejected.
type AppliedJob struct {
	CompanyName string
	AppliedAt   time.Time
	RejectedAt  *time.Time
}

// AppliedJobs lists the user's applications. Jobs at new, viewed or shortlisted are
// left out unless applied_at shows they were applied to before reaching that status,
// such as an application later moved to shortlisted. Rejected jobs that predate applied_at have
// none, so created_at stands in for it; the two fall on the same day for jobs that
// have both.
func (r *DiscoveredJobRepository) AppliedJobs(userID string) ([]AppliedJob, error) {
	var rows []AppliedJob
	if err := r.db.Model(&domain.Job{}).
		Select("company_name, COALESCE(applied_at, created_at) AS applied_at, rejected_at").
		Where("user_id = ? AND company_name <> '' AND (applied_at IS NOT NULL OR status NOT IN ?)", userID, []domain.JobStatus{
			domain.StatusNew, domain.StatusViewed, domain.StatusShortlisted,
		}).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
