package fit

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"job-tracker-backend/internal/domain"
)

func TestJobIDs(t *testing.T) {
	linkedIn := map[string]string{
		"https://www.linkedin.com/jobs/view/4483536053/":                            "4483536053",
		"https://au.linkedin.com/jobs/view/senior-engineer-at-acme-4483536053?x=1":  "4483536053",
		"https://www.linkedin.com/jobs/search/?currentJobId=4482906660&keywords=go": "4482906660",
		"https://www.linkedin.com/feed/":                                            "",
	}
	for raw, want := range linkedIn {
		u, _ := url.Parse(raw)
		if got := linkedInJobID(u); got != want {
			t.Errorf("linkedInJobID(%s) = %q, want %q", raw, got, want)
		}
	}

	seek := map[string]string{
		"https://www.seek.com.au/job/95489437":                      "95489437",
		"https://au.seek.com/job/95489437?ref=recom":                "95489437",
		"https://www.seek.com.au/software-jobs?jobId=95483171&type": "95483171",
		"https://www.seek.com.au/software-jobs":                     "",
	}
	for raw, want := range seek {
		u, _ := url.Parse(raw)
		if got := seekJobID(u); got != want {
			t.Errorf("seekJobID(%s) = %q, want %q", raw, got, want)
		}
	}
}

func TestHTMLToText(t *testing.T) {
	got := htmlToText(`<p><strong>About us</strong></p><script>track()</script>
		<ul><li>Go &amp; Postgres</li><li>Hybrid,&nbsp;2 days</li></ul><br><p>Apply   now</p>`)
	want := "About us\n\n- Go & Postgres\n\n- Hybrid, 2 days\n\nApply now"
	if got != want {
		t.Errorf("htmlToText:\n got %q\nwant %q", got, want)
	}
}

func TestPublicAddressesOnly(t *testing.T) {
	for _, address := range []string{"127.0.0.1:80", "10.43.0.10:8080", "192.168.1.5:443", "169.254.169.254:80", "100.101.102.103:443", "[::1]:80"} {
		if err := publicAddressesOnly("tcp", address, nil); err == nil {
			t.Errorf("%s should be refused, got %v", address, err)
		}
	}
	if err := publicAddressesOnly("tcp", "13.107.42.14:443", nil); err != nil {
		t.Errorf("a public address should be allowed, got %v", err)
	}
}

// fakeClaude writes a stand-in for the CLI that prints the given JSON.
func fakeClaude(t *testing.T, output string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude")
	script := "#!/bin/sh\ncat > /dev/null\ncat <<'EOF'\n" + output + "\nEOF\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScorer(t *testing.T) {
	posting := &domain.FitPosting{JobTitle: "Backend Engineer", Source: "manual"}

	ok := fakeClaude(t, `{"type":"result","is_error":false,"result":"","structured_output":{"score":4,"summary":"Good.","strengths":["Go"],"gaps":[],"flags":[]}}`)
	fit, err := NewScorer(ok, "claude-opus-5-5", "low").Score(context.Background(), "profile", posting)
	if err != nil || fit.Score != 4 || fit.Model != "claude-opus-5-5" || fit.Strengths[0] != "Go" {
		t.Fatalf("unexpected fit %+v, err %v", fit, err)
	}

	failing := fakeClaude(t, `{"type":"result","subtype":"success","is_error":true,"result":"Invalid API key · Please run /login"}`)
	if _, err := NewScorer(failing, "m", "low").Score(context.Background(), "profile", posting); err == nil || !strings.Contains(err.Error(), "Please run /login") {
		t.Errorf("an error result should surface its message, got %v", err)
	}

	outOfRange := fakeClaude(t, `{"is_error":false,"structured_output":{"score":7,"summary":"","strengths":[],"gaps":[],"flags":[]}}`)
	if _, err := NewScorer(outOfRange, "m", "low").Score(context.Background(), "profile", posting); err == nil {
		t.Error("a score of 7 should be rejected")
	}

	if _, err := NewScorer("/nonexistent/claude", "m", "low").Score(context.Background(), "profile", posting); !errors.Is(err, ErrUnavailable) {
		t.Errorf("a missing CLI should be ErrUnavailable, got %v", err)
	}
}

func TestFormatPosting(t *testing.T) {
	got := FormatPosting(&domain.FitPosting{JobTitle: "Dev", Source: "web"})
	if !strings.Contains(got, "Company: unknown") || !strings.Contains(got, "Description: not available") {
		t.Errorf("missing fields should read as unknown:\n%s", got)
	}
}
