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

	// Preload attachments for each job
	if len(jobs) > 0 {
		jobIDs := make([]string, len(jobs))
		for i, job := range jobs {
			jobIDs[i] = job.ID
		}
		var attachments []domain.Attachment
		if err := r.db.Where("job_id IN ?", jobIDs).Find(&attachments).Error; err != nil {
			return nil, err
		}
		// Attachments to jobs
		attachmentMap := make(map[string][]domain.Attachment)
		for _, att := range attachments {
			attachmentMap[att.JobID] = append(attachmentMap[att.JobID], att)
		}
		for i := range jobs {
			jobs[i].Attachments = attachmentMap[jobs[i].ID]
		}
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

// Attachment methods

func (r *JobRepository) CreateAttachment(attachment *domain.Attachment) error {
	return r.db.Create(attachment).Error
}

func (r *JobRepository) GetAttachmentByID(id string) (*domain.Attachment, error) {
	var attachment domain.Attachment
	if err := r.db.First(&attachment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appErrors.ErrNotFound
		}
		return nil, err
	}
	return &attachment, nil
}

func (r *JobRepository) GetAttachmentsByJobID(jobID string) ([]domain.Attachment, error) {
	var attachments []domain.Attachment
	if err := r.db.Where("job_id = ?", jobID).Order("created_at DESC").Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *JobRepository) DeleteAttachment(id string) error {
	result := r.db.Delete(&domain.Attachment{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return appErrors.ErrNotFound
	}
	return nil
}

func (r *JobRepository) DeleteAttachmentsByJobID(jobID string) error {
	return r.db.Where("job_id = ?", jobID).Delete(&domain.Attachment{}).Error
}
