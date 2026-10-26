package main

import (
	"strings"
	"testing"
	"time"

	"job-tracker-backend/internal/domain"
)

func TestGroupByRole(t *testing.T) {
	roles := groupByRole([]domain.UnscoredDiscoveredJob{
		{ID: "1", MatchKey: "acme|dev", Source: "linkedin", Description: "short"},
		{ID: "2", MatchKey: "globex|dev"},
		{ID: "3", MatchKey: "acme|dev", Source: "seek", Description: "a longer description"},
	})
	if len(roles) != 2 {
		t.Fatalf("want 2 roles, got %d", len(roles))
	}
	if strings.Join(roles[0].ids, ",") != "1,3" || roles[0].job.Source != "seek" {
		t.Errorf("same role should share ids and score the fuller copy, got %+v", roles[0])
	}
}

func TestFormatTelegram(t *testing.T) {
	now := time.Date(2026, 10, 27, 9, 30, 0, 0, time.UTC)
	role := func(score int, company string) scoredRole {
		return scoredRole{
			job: domain.UnscoredDiscoveredJob{CompanyName: company, JobTitle: "Engineer", JobURL: "https://example.com/" + company},
			fit: &domain.Fit{Score: score, Summary: company + " summary.", Flags: nil},
		}
	}

	if got := formatTelegram(nil, nil, now, ""); got != "" {
		t.Errorf("nothing scored and no warnings should send nothing, got %q", got)
	}

	got := formatTelegram([]scoredRole{role(4, "Acme"), role(2, "Initech"), role(5, "Globex"), role(3, "Hooli")}, nil, now, "https://tracker/discovered")
	if !strings.Contains(got, "Scored 4 roles: 2 good or better, 1 possible, 1 weak.") {
		t.Errorf("missing counts:\n%s", got)
	}
	if strings.Index(got, "Globex") > strings.Index(got, "Acme") {
		t.Errorf("5/5 should come before 4/5:\n%s", got)
	}
	if strings.Contains(got, "Initech") || strings.Contains(got, "Hooli") {
		t.Errorf("only scores of 4 and up should be listed:\n%s", got)
	}

	weakOnly := formatTelegram([]scoredRole{role(2, "Initech")}, []string{"one failed"}, now, "")
	if !strings.Contains(weakOnly, "No strong fits today.") || !strings.Contains(weakOnly, "• one failed") {
		t.Errorf("unexpected message:\n%s", weakOnly)
	}
}
