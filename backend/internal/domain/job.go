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
	ID          string       `json:"id" gorm:"primaryKey;type:varchar(36)"`
	JobTitle    string       `json:"job_title" gorm:"not null;type:varchar(500)"`
	CompanyName string       `json:"company_name" gorm:"type:varchar(500)"`
	Location    string       `json:"location" gorm:"type:varchar(500)"`
	JobURL      string       `json:"job_url" gorm:"type:varchar(2000)"`
	Description string       `json:"description" gorm:"type:text"`
	Salary      string       `json:"salary" gorm:"type:varchar(200)"`
	JobType     string       `json:"job_type" gorm:"type:varchar(100)"`
	IsRemote    bool         `json:"is_remote" gorm:"default:false"`
	Source      string       `json:"source" gorm:"type:varchar(100)"`
	Status      string       `json:"status" gorm:"default:'new';type:varchar(50);index"`
	Notes       string       `json:"notes" gorm:"type:text"`
	Attachments []Attachment `json:"attachments" gorm:"foreignKey:JobID"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
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
	JobTitle    string `json:"job_title" binding:"required"`
	CompanyName string `json:"company_name"`
	Location    string `json:"location"`
	JobURL      string `json:"job_url" binding:"required,url"`
	Description string `json:"description"`
	Salary      string `json:"salary"`
	JobType     string `json:"job_type"`
	IsRemote    bool   `json:"is_remote"`
	Source      string `json:"source"`
	Notes       string `json:"notes"`
}

type JobUpdateInput struct {
	JobTitle    string `json:"job_title"`
	CompanyName string `json:"company_name"`
	Location    string `json:"location"`
	JobURL      string `json:"job_url"`
	Description string `json:"description"`
	Salary      string `json:"salary"`
	JobType     string `json:"job_type"`
	IsRemote    bool   `json:"is_remote"`
	Source      string `json:"source"`
	Status      string `json:"status"`
	Notes       string `json:"notes"`
}

type JobStatusUpdate struct {
	Status string `json:"status" binding:"required"`
}

type JobFilter struct {
	Status string `query:"status"`
	Source string `query:"source"`
}

// Attachment represents a file attachment (resume/cover letter) for a job
type Attachment struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	JobID     string    `json:"job_id" gorm:"type:varchar(36);index;not null"`
	FileName  string    `json:"file_name" gorm:"type:varchar(255);not null"`
	FileType  string    `json:"file_type" gorm:"type:varchar(50)"` // "resume" | "cover_letter"
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
