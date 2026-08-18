package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// How a posting is applied to, as advertised by the job board.
const (
	ApplyTypeEasyApply = "easy_apply"
	ApplyTypeExternal  = "external"
	ApplyTypeUnknown   = "unknown"
)

// DiscoveredRetentionDays bounds how much scrape history is kept. Discovered jobs
// are a reading surface, not a record: anything older than this is pruned on the
// next ingest.
const DiscoveredRetentionDays = 7

// DiscoveredJob is a posting surfaced by an automated scrape. It is deliberately
// separate from Job: Job means "a role I am tracking an application for", and the
// scrape must not pollute that — both the scraper's own de-duplication and the
// resume workflow ask the jobs table "have I applied at this company before?".
type DiscoveredJob struct {
	ID     string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID string `json:"user_id" gorm:"type:varchar(36);index;uniqueIndex:idx_discovered_user_external"`
	// ExternalID is the job board's own id (for LinkedIn, the jobPosting urn digits).
	// Unique per user so a re-scrape updates the existing row instead of duplicating it.
	ExternalID    string    `json:"external_id" gorm:"type:varchar(64);uniqueIndex:idx_discovered_user_external"`
	JobTitle      string    `json:"job_title" gorm:"not null;type:varchar(500)"`
	CompanyName   string    `json:"company_name" gorm:"type:varchar(500)"`
	Location      string    `json:"location" gorm:"type:varchar(500)"`
	JobURL        string    `json:"job_url" gorm:"type:varchar(2000)"`
	Source        string    `json:"source" gorm:"type:varchar(100)"`
	PostedDate    string    `json:"posted_date" gorm:"type:varchar(20)"`
	ApplyType     string    `json:"apply_type" gorm:"type:varchar(20);default:'unknown'"`
	AppliedBefore bool      `json:"applied_before" gorm:"default:false"`
	Dismissed     bool      `json:"dismissed" gorm:"default:false;index"`
	DiscoveredAt  time.Time `json:"discovered_at" gorm:"index"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (d *DiscoveredJob) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	if d.ApplyType == "" {
		d.ApplyType = ApplyTypeUnknown
	}
	if d.DiscoveredAt.IsZero() {
		d.DiscoveredAt = time.Now()
	}
	return nil
}

// DiscoveredJobInput is one scraped posting as submitted by a scraper.
type DiscoveredJobInput struct {
	ExternalID  string `json:"external_id" binding:"required"`
	JobTitle    string `json:"job_title" binding:"required"`
	CompanyName string `json:"company_name"`
	Location    string `json:"location"`
	JobURL      string `json:"job_url"`
	Source      string `json:"source"`
	PostedDate  string `json:"posted_date"`
	ApplyType   string `json:"apply_type"`
}

// DiscoveredIngestInput is a whole scrape run's worth of postings.
type DiscoveredIngestInput struct {
	Jobs []DiscoveredJobInput `json:"jobs"`
}

// DiscoveredIngestResult reports what an ingest did, so a scraper's logs are
// meaningful without querying afterwards.
type DiscoveredIngestResult struct {
	Received int `json:"received"`
	Created  int `json:"created"`
	Updated  int `json:"updated"`
	Pruned   int `json:"pruned"`
}
