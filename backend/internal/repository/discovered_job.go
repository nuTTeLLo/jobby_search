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
