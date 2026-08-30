package repository

import (
	"errors"
	"strings"

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

func (r *JobRepository) GetByID(id, userID string) (*domain.Job, error) {
	var job domain.Job
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appErrors.ErrNotFound
		}
		return nil, err
	}
	return &job, nil
}

func (r *JobRepository) GetAll(filter *domain.JobFilter) ([]domain.Job, int64, error) {
	var jobs []domain.Job
	query := r.db.Model(&domain.Job{})

	if filter != nil {
		if filter.UserID != "" {
			query = query.Where("user_id = ?", filter.UserID)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.Source != "" {
			query = query.Where("source = ?", filter.Source)
		}
		// Free-text search: every term must match one of the listed columns.
		// Backslash is Postgres' default LIKE escape character, so escapeLike's
		// output needs no explicit ESCAPE clause.
		for _, term := range strings.Fields(filter.Search) {
			pattern := "%" + escapeLike(term) + "%"
			query = query.Where(
				"(job_title ILIKE ? OR company_name ILIKE ? OR location ILIKE ? OR job_type ILIKE ? OR source ILIKE ? OR status ILIKE ?)",
				pattern, pattern, pattern, pattern, pattern, pattern,
			)
		}
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// id is a tiebreaker so rows can't shift between pages when two jobs share
	// a created_at.
	query = query.Order("created_at DESC, id DESC")
	if filter != nil {
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	if err := query.Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	// Preload attachments for each job. Skip the `data` column — the blobs are
	// never serialized in a list response and loading them is expensive.
	if len(jobs) > 0 {
		jobIDs := make([]string, len(jobs))
		for i, job := range jobs {
			jobIDs[i] = job.ID
		}
		var attachments []domain.Attachment
		if err := r.db.
			Select("id", "job_id", "file_name", "file_type", "mime_type", "file_size", "created_at").
			Where("job_id IN ?", jobIDs).Find(&attachments).Error; err != nil {
			return nil, 0, err
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

	return jobs, total, nil
}

// escapeLike neutralises LIKE wildcards so a user's search text is matched
// literally.
func escapeLike(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return r.Replace(s)
}

func (r *JobRepository) Update(job *domain.Job) error {
	return r.db.Save(job).Error
}

func (r *JobRepository) Delete(id, userID string) error {
	result := r.db.Delete(&domain.Job{}, "id = ? AND user_id = ?", id, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return appErrors.ErrNotFound
	}
	return nil
}

func (r *JobRepository) ExistsByURL(url, userID string) (bool, error) {
	var count int64
	if err := r.db.Model(&domain.Job{}).Where("job_url = ? AND user_id = ?", url, userID).Count(&count).Error; err != nil {
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
