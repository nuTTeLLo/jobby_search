package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"job-tracker-backend/internal/auth"
	"job-tracker-backend/internal/service"
)

func newTestAuthHandler() http.Handler {
	providers := auth.NewProviders("https://tracker.example", "gid", "gsecret", "", "")
	svc := service.NewAuthService(nil, providers, []string{"owner@example.com"}, "secret", time.Hour)
	return NewAuthHandler(svc, "https://app.example", true).PublicRoutes()
}

func TestStartRedirectsToProvider(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestAuthHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/google/start", nil))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d", rec.Code)
	}
	loc, _ := url.Parse(rec.Header().Get("Location"))
	q := loc.Query()
	if loc.Host != "accounts.google.com" || q.Get("redirect_uri") != "https://tracker.example/api/auth/google/callback" {
		t.Errorf("redirect = %s", loc)
	}
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Error("missing PKCE challenge")
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("cookies = %+v", cookies)
	}
	state, _, _ := strings.Cut(cookies[0].Value, ".")
	if state == "" || state != q.Get("state") {
		t.Errorf("cookie state %q != redirect state %q", state, q.Get("state"))
	}
}

func TestStartUnknownProvider(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestAuthHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/github/start", nil))
	if got := rec.Header().Get("Location"); got != "https://app.example/login?error=unknown_provider" {
		t.Errorf("Location = %q", got)
	}
}

// Each of these must bounce back to the login page before any code exchange.
func TestCallbackRejects(t *testing.T) {
	cases := []struct {
		name, query, cookie, wantError string
	}{
		{"user cancelled", "error=access_denied&state=s", "s.v", "cancelled"},
		{"no cookie", "code=c&state=s", "", "expired"},
		{"state mismatch", "code=c&state=attacker", "s.v", "state_mismatch"},
		{"empty state", "code=c", "s.v", "state_mismatch"},
		{"malformed cookie", "code=c&state=s", "s", "state_mismatch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/google/callback?"+tc.query, nil)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: oauthCookie, Value: tc.cookie})
			}
			rec := httptest.NewRecorder()
			newTestAuthHandler().ServeHTTP(rec, req)

			if got, want := rec.Header().Get("Location"), "https://app.example/login?error="+tc.wantError; got != want {
				t.Errorf("Location = %q, want %q", got, want)
			}
			if c := rec.Result().Cookies(); len(c) != 1 || c[0].MaxAge >= 0 {
				t.Errorf("flow cookie not cleared: %+v", c)
			}
		})
	}
}
