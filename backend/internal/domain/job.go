package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JobStatus string

const (
	StatusNew         JobStatus = "new"
	StatusViewed      JobStatus = "viewed"
	StatusApplied     JobStatus = "applied"
	StatusRejected    JobStatus = "rejected"
	StatusShortlisted JobStatus = "shortlisted"
)

type Job struct {
	ID           string       `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID       string       `json:"user_id" gorm:"type:varchar(36);index;index:idx_jobs_user_created,priority:1"`
	JobTitle     string       `json:"job_title" gorm:"not null;type:varchar(500)"`
	CompanyName  string       `json:"company_name" gorm:"type:varchar(500)"`
	Location     string       `json:"location" gorm:"type:varchar(500)"`
	JobURL       string       `json:"job_url" gorm:"type:varchar(2000)"`
	Description  string       `json:"description" gorm:"type:text"`
	Salary       string       `json:"salary" gorm:"type:varchar(200)"`
	JobType      string       `json:"job_type" gorm:"type:varchar(100)"`
	IsRemote     bool         `json:"is_remote" gorm:"default:false"`
	EasyApply    bool         `json:"easy_apply" gorm:"default:false"`
	ViaRecruiter bool         `json:"via_recruiter" gorm:"default:false"`
	Source       string       `json:"source" gorm:"type:varchar(100)"`
	Status       string       `json:"status" gorm:"default:'new';type:varchar(50);index"`
	Notes        string       `json:"notes" gorm:"type:text"`
	Attachments  []Attachment `json:"attachments" gorm:"foreignKey:JobID;constraint:OnDelete:CASCADE"`
	// Paired with UserID in idx_jobs_user_created: every list query filters by
	// user and orders by created_at.
	CreatedAt time.Time `json:"created_at" gorm:"index:idx_jobs_user_created,priority:2,sort:desc"`
	UpdatedAt time.Time `json:"updated"`
}

func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.New().String()
	}
	if j.Status == "" {
		j.Status = string(StatusNew)
	}
	return nil
}

type JobCreateInput struct {
	JobTitle     string `json:"job_title" binding:"required"`
	CompanyName  string `json:"company_name"`
	Location     string `json:"location"`
	JobURL       string `json:"job_url" binding:"required,url"`
	Description  string `json:"description"`
	Salary       string `json:"salary"`
	JobType      string `json:"job_type"`
	IsRemote     bool   `json:"is_remote"`
	EasyApply    bool   `json:"easy_apply"`
	ViaRecruiter *bool  `json:"via_recruiter"`
	Source       string `json:"source"`
	Notes        string `json:"notes"`
}

type JobUpdateInput struct {
	JobTitle     string `json:"job_title"`
	CompanyName  string `json:"company_name"`
	Location     string `json:"location"`
	JobURL       string `json:"job_url"`
	Description  string `json:"description"`
	Salary       string `json:"salary"`
	JobType      string `json:"job_type"`
	IsRemote     bool   `json:"is_remote"`
	EasyApply    bool   `json:"easy_apply"`
	ViaRecruiter *bool  `json:"via_recruiter"`
	Source       string `json:"source"`
	Status       string `json:"status"`
	Notes        string `json:"notes"`
}

type JobStatusUpdate struct {
	Status string `json:"status" binding:"required"`
}

type JobFilter struct {
	Status string `query:"status"`
	Source string `query:"source"`
	// Search is a free-text query matched against the fields shown in the
	// tracked jobs table. Whitespace-separated terms are ANDed together.
	Search string `query:"q"`
	UserID string
	Limit  int
	Offset int
}

// JobPage is one page of tracked jobs plus the total matching the filter,
// so the client can render pagination controls.
type JobPage struct {
	Jobs     []Job `json:"jobs"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// Attachment represents a document attached to a job — a resume, a cover
// letter, or typed-out text such as question responses.
type Attachment struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	JobID     string    `json:"job_id" gorm:"type:varchar(36);index;not null"`
	FileName  string    `json:"file_name" gorm:"type:varchar(255);not null"`
	FileType  string    `json:"file_type" gorm:"type:varchar(50)"` // see service.allowedFileTypes
	MIMEType  string    `json:"mime_type" gorm:"type:varchar(100)"`
	Data      []byte    `json:"-" gorm:"type:bytea"` // Don't serialize to JSON
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}
