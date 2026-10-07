package service

import (
	"errors"
	"testing"
	"time"

	"job-tracker-backend/internal/auth"
	"job-tracker-backend/internal/domain"
	appErrors "job-tracker-backend/pkg/errors"
)

type fakeUsers struct{ byEmail map[string]*domain.User }

func (f *fakeUsers) Create(u *domain.User) error {
	u.ID = "new-" + u.Email
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUsers) GetByEmail(email string) (*domain.User, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, appErrors.ErrNotFound
}

func (f *fakeUsers) GetByID(id string) (*domain.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, appErrors.ErrNotFound
}

func TestLoginByEmail(t *testing.T) {
	users := &fakeUsers{byEmail: map[string]*domain.User{
		"owner@example.com": {ID: "existing-id", Email: "owner@example.com"},
	}}
	svc := NewAuthService(users, nil, []string{" Owner@Example.com", "friend@example.com", ""}, "secret", time.Hour)

	t.Run("existing account keeps its ID", func(t *testing.T) {
		resp, err := svc.loginByEmail([]string{"OWNER@example.com "})
		if err != nil {
			t.Fatal(err)
		}
		if resp.User.ID != "existing-id" {
			t.Errorf("got user %q, want existing-id", resp.User.ID)
		}
		claims, err := auth.ValidateToken(resp.Token, "secret")
		if err != nil || claims.UserID != "existing-id" || claims.Email != "owner@example.com" {
			t.Errorf("token claims = %+v, %v", claims, err)
		}
	})

	t.Run("allowed newcomer gets an account", func(t *testing.T) {
		resp, err := svc.loginByEmail([]string{"friend@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.User.ID != "new-friend@example.com" {
			t.Errorf("got user %q", resp.User.ID)
		}
	})

	t.Run("first allowed verified email wins", func(t *testing.T) {
		resp, err := svc.loginByEmail([]string{"work@corp.example", "owner@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.User.ID != "existing-id" {
			t.Errorf("got user %q, want existing-id", resp.User.ID)
		}
	})

	t.Run("email not on the allowlist is refused", func(t *testing.T) {
		for _, email := range []string{"stranger@example.com", ""} {
			if _, err := svc.loginByEmail([]string{email}); !errors.Is(err, ErrEmailNotAllowed) {
				t.Errorf("%q: err = %v, want ErrEmailNotAllowed", email, err)
			}
		}
		if _, ok := users.byEmail["stranger@example.com"]; ok {
			t.Error("refused email still created a user")
		}
	})
}

func TestGmailAliasMatchesExistingAccount(t *testing.T) {
	users := &fakeUsers{byEmail: map[string]*domain.User{
		"nuttello@gmail.com": {ID: "owner-id", Email: "nuttello@gmail.com"},
	}}
	svc := NewAuthService(users, nil, []string{"nuttello@gmail.com"}, "secret", time.Hour)

	for _, alias := range []string{"nuttello+dev@gmail.com", "Nut.Tello@googlemail.com"} {
		resp, err := svc.loginByEmail([]string{alias})
		if err != nil {
			t.Fatalf("%s: %v", alias, err)
		}
		if resp.User.ID != "owner-id" {
			t.Errorf("%s: got user %q, want owner-id", alias, resp.User.ID)
		}
	}
	if len(users.byEmail) != 1 {
		t.Errorf("an alias created a second account: %v", users.byEmail)
	}
}

func TestCanonicalEmail(t *testing.T) {
	cases := map[string]string{
		"nuttello+dev@gmail.com":     "nuttello@gmail.com",
		" Nut.Tello@GoogleMail.com ": "nuttello@gmail.com",
		"first.last+x@example.com":   "first.last+x@example.com", // only Gmail folds aliases
		"not-an-email":               "not-an-email",
	}
	for in, want := range cases {
		if got := canonicalEmail(in); got != want {
			t.Errorf("canonicalEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
