package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"job-tracker-backend/internal/auth"
	"job-tracker-backend/internal/domain"
	appErrors "job-tracker-backend/pkg/errors"

	"golang.org/x/oauth2"
)

// ErrEmailNotAllowed means the provider vouched for the email but it isn't on
// ALLOWED_EMAILS. Anyone with a Google or GitHub account can finish OAuth, so
// signing in proves who someone is, not that they may use the tracker.
var ErrEmailNotAllowed = errors.New("email not allowed")

// userStore is the slice of UserRepository the auth service uses.
type userStore interface {
	Create(user *domain.User) error
	GetByEmail(email string) (*domain.User, error)
	GetByID(id string) (*domain.User, error)
	ListGmailUsers() ([]domain.User, error)
}

type AuthService struct {
	userRepo      userStore
	providers     map[string]*auth.Provider
	allowedEmails map[string]bool
	jwtSecret     string
	jwtExpiry     time.Duration
}

func NewAuthService(repo userStore, providers map[string]*auth.Provider, allowedEmails []string, secret string, expiry time.Duration) *AuthService {
	allowed := map[string]bool{}
	for _, e := range allowedEmails {
		if e = canonicalEmail(e); e != "" {
			allowed[e] = true
		}
	}
	return &AuthService{userRepo: repo, providers: providers, allowedEmails: allowed, jwtSecret: secret, jwtExpiry: expiry}
}

// AuthURL is the provider's sign-in page to send the browser to. state comes
// back unchanged on the callback; the PKCE verifier stays on our side and only
// its hash goes to the provider, so a stolen code can't be redeemed without it.
func (s *AuthService) AuthURL(provider, state, verifier string) (string, error) {
	p, ok := s.providers[provider]
	if !ok {
		return "", auth.ErrUnknownProvider
	}
	return p.Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier)), nil
}

// CompleteLogin finishes the flow: trade the one-time code (plus client secret
// and PKCE verifier) for an access token, use it once to learn the verified
// email, then sign the user in with our own JWT exactly as a password login did.
func (s *AuthService) CompleteLogin(ctx context.Context, provider, code, verifier string) (*domain.AuthResponse, error) {
	p, ok := s.providers[provider]
	if !ok {
		return nil, auth.ErrUnknownProvider
	}
	token, err := p.Config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	emails, err := p.FetchEmails(ctx, p.Config.Client(ctx, token))
	if err != nil {
		return nil, fmt.Errorf("fetch email: %w", err)
	}
	return s.loginByEmail(emails)
}

// loginByEmail signs in with the first of the provider's verified emails that is
// allowed. It matches users on the canonical email, so an account created before
// OAuth keeps its ID — and with it its jobs and the scraper's tracker_user_id —
// whichever Gmail alias a provider reports.
func (s *AuthService) loginByEmail(emails []string) (*domain.AuthResponse, error) {
	email := ""
	for _, e := range emails {
		if c := canonicalEmail(e); s.allowedEmails[c] {
			email = c
			break
		}
	}
	if email == "" {
		return nil, ErrEmailNotAllowed
	}

	user, err := s.findUser(email)
	if errors.Is(err, appErrors.ErrNotFound) {
		user = &domain.User{Email: email}
		err = s.userRepo.Create(user)
	}
	if err != nil {
		return nil, err
	}
	return s.buildAuthResponse(user)
}

// findUser looks up the user for a canonical email. Rows from before OAuth
// stored the address as typed, so a Gmail account may sit under an alias
// (nut.tello+x@googlemail.com); fall back to comparing canonical forms rather
// than create a second account for the same inbox.
func (s *AuthService) findUser(email string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(email)
	if !errors.Is(err, appErrors.ErrNotFound) || !strings.HasSuffix(email, "@gmail.com") {
		return user, err
	}
	candidates, err := s.userRepo.ListGmailUsers()
	if err != nil {
		return nil, err
	}
	for i := range candidates {
		if canonicalEmail(candidates[i].Email) == email {
			return &candidates[i], nil
		}
	}
	return nil, appErrors.ErrNotFound
}

func (s *AuthService) Me(userID string) (*domain.AuthUser, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	return &domain.AuthUser{ID: user.ID, Email: user.Email}, nil
}

func (s *AuthService) buildAuthResponse(user *domain.User) (*domain.AuthResponse, error) {
	token, err := auth.GenerateToken(user.ID, user.Email, s.jwtSecret, s.jwtExpiry)
	if err != nil {
		return nil, err
	}
	return &domain.AuthResponse{
		Token: token,
		User:  domain.AuthUser{ID: user.ID, Email: user.Email},
	}, nil
}

// canonicalEmail lower-cases an address and folds Gmail aliases into the base
// inbox: Gmail ignores dots and anything after a "+" in the local part, so
// nut.tello+dev@gmail.com is the same mailbox as nuttello@gmail.com. Only Gmail
// guarantees that, so other domains are left alone.
func canonicalEmail(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	local, domain, ok := strings.Cut(email, "@")
	if !ok || (domain != "gmail.com" && domain != "googlemail.com") {
		return email
	}
	local, _, _ = strings.Cut(local, "+")
	return strings.ReplaceAll(local, ".", "") + "@gmail.com"
}
