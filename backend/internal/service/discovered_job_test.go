package service

import (
	"testing"

	"job-tracker-backend/internal/domain"
)

func TestMatchKey(t *testing.T) {
	key := func(company, title string) string {
		return matchKey(&domain.DiscoveredJob{ID: company + title, CompanyName: company, JobTitle: title})
	}

	same := []struct{ a, b [2]string }{
		{[2]string{"Acme Pty Ltd", "Senior Full-Stack Engineer"}, [2]string{"acme", "Senior Full Stack Engineer"}},
		{[2]string{"Motorola Solutions", "Senior Fullstack Softare Engineer (Web)"}, [2]string{"Motorola Solutions", "Senior Fullstack Softare Engineer (Web)"}},
		{[2]string{"REA Group", "Software Engineer"}, [2]string{"REA", "software engineer"}},
		// Seen in a real run: LinkedIn and Indeed name the bank differently.
		{[2]string{"Commonwealth Bank", "Staff Software Engineer (AWS and AI)"}, [2]string{"Commonwealth Bank of Australia", "Staff Software Engineer (AWS and AI)"}},
		// Seek and LinkedIn: a brand alias on one side, a region suffix on the other.
		{[2]string{"Allume Energy", "Data Engineer"}, [2]string{"Allume ANZ", "Data Engineer"}},
		{[2]string{"Acme NZ", "Data Engineer"}, [2]string{"Acme", "Data Engineer"}},
		{[2]string{"Rauland Australia and New Zealand", "Data Engineer"}, [2]string{"Rauland", "Data Engineer"}},
	}
	for _, tc := range same {
		if ka, kb := key(tc.a[0], tc.a[1]), key(tc.b[0], tc.b[1]); ka != kb {
			t.Errorf("%v and %v should match: %q != %q", tc.a, tc.b, ka, kb)
		}
	}

	different := []struct{ a, b [2]string }{
		{[2]string{"REA Group", "Software Engineer"}, [2]string{"REA Group", "Software Engineer - Mobile"}},
		{[2]string{"Acme", "Data Engineer"}, [2]string{"Globex", "Data Engineer"}},
		// No company to match on: each posting stands alone.
		{[2]string{"", "Data Engineer"}, [2]string{"", "Data Engineer "}},
	}
	for _, tc := range different {
		if ka, kb := key(tc.a[0], tc.a[1]), key(tc.b[0], tc.b[1]); ka == kb {
			t.Errorf("%v and %v should not match, both %q", tc.a, tc.b, ka)
		}
	}
}
