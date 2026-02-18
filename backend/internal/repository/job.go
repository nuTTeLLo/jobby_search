package repository

import (
	"errors"
	"job-tracker-backend/internal/domain"
	appErrors "job-tracker-backend/pkg/errors"

	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(job *domain.Job) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) GetByID(id string) (*domain.Job, error) {
	var job domain.Job
	if err := r.db.First(&job, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appErrors.ErrNotFound
		}
		return nil, err
	}
	return &job, nil
}

func (r *JobRepository) GetAll(filter *domain.JobFilter) ([]domain.Job, error) {
	var jobs []domain.Job
	query := r.db.Model(&domain.Job{})

	if filter != nil {
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.Source != "" {
			query = query.Where("source = ?", filter.Source)
		}
	}

	if err := query.Order("created_at DESC").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *JobRepository) Update(job *domain.Job) error {
	return r.db.Save(job).Error
}

func (r *JobRepository) Delete(id string) error {
	result := r.db.Delete(&domain.Job{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return appErrors.ErrNotFound
	}
	return nil
}

func (r *JobRepository) ExistsByURL(url string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.Job{}).Where("job_url = ?", url).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *JobRepository) CreateBatch(jobs []domain.Job) error {
	if len(jobs) == 0 {
		return nil
	}
	return r.db.Create(&jobs).Error
}
