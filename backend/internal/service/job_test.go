package service

import (
	"testing"
	"time"

	"job-tracker-backend/internal/domain"
)

func TestStampRejected(t *testing.T) {
	job := &domain.Job{Status: string(domain.StatusApplied)}
	stampRejected(job)
	if job.RejectedAt != nil {
		t.Fatalf("applied job stamped: %v", job.RejectedAt)
	}

	job.Status = string(domain.StatusRejected)
	stampRejected(job)
	if job.RejectedAt == nil {
		t.Fatal("rejected job not stamped")
	}

	// A repeated rejected status must not restart the clock.
	first := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	job.RejectedAt = &first
	stampRejected(job)
	if !job.RejectedAt.Equal(first) {
		t.Errorf("stamp moved to %v", job.RejectedAt)
	}

	// A revived application stands rejected no longer.
	job.Status = string(domain.StatusInterview)
	stampRejected(job)
	if job.RejectedAt != nil {
		t.Errorf("revived job still stamped: %v", job.RejectedAt)
	}
}
