package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

var (
	ErrUnknownProvider = errors.New("unknown oauth provider")
	ErrNoVerifiedEmail = errors.New("provider returned no verified email")
)

// Provider is one "Sign in with …" option. Config holds the client ID/secret and
// the provider's URLs; FetchEmails turns the access token from the code exchange
// into the user's verified emails — the only thing the tracker needs from them.
// The preferred (primary) address comes first.
type Provider struct {
	Config      *oauth2.Config
	FetchEmails func(ctx context.Context, client *http.Client) ([]string, error)
}

// NewProviders builds the providers that have credentials configured, keyed by
// the name used in /api/auth/{provider}/... routes. publicBaseURL is where the
// browser reaches the backend; the callback URL derived from it must match the
// one registered with the provider exactly.
func NewProviders(publicBaseURL, googleID, googleSecret, githubID, githubSecret string) map[string]*Provider {
	base := strings.TrimRight(publicBaseURL, "/")
	providers := map[string]*Provider{}
	if googleID != "" {
		providers["google"] = &Provider{
			Config: &oauth2.Config{
				ClientID:     googleID,
				ClientSecret: googleSecret,
				Endpoint:     endpoints.Google,
				RedirectURL:  base + "/api/auth/google/callback",
				Scopes:       []string{"openid", "email"},
			},
			FetchEmails: googleEmails,
		}
	}
	if githubID != "" {
		providers["github"] = &Provider{
			Config: &oauth2.Config{
				ClientID:     githubID,
				ClientSecret: githubSecret,
				Endpoint:     endpoints.GitHub,
				RedirectURL:  base + "/api/auth/github/callback",
				Scopes:       []string{"user:email"},
			},
			FetchEmails: githubEmails,
		}
	}
	return providers
}

func googleEmails(ctx context.Context, client *http.Client) ([]string, error) {
	var info struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := getJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &info); err != nil {
		return nil, err
	}
	if info.Email == "" || !info.EmailVerified {
		return nil, ErrNoVerifiedEmail
	}
	return []string{info.Email}, nil
}

// githubEmails reads /user/emails rather than /user: the profile email is
// optional and unverified, while this list says which addresses are verified.
// All of them are returned, primary first, so a secondary address can still
// match an allowed account.
func githubEmails(ctx context.Context, client *http.Client) ([]string, error) {
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
		return nil, err
	}
	var verified []string
	for _, e := range emails {
		switch {
		case !e.Verified:
		case e.Primary:
			verified = append([]string{e.Email}, verified...)
		default:
			verified = append(verified, e.Email)
		}
	}
	if len(verified) == 0 {
		return nil, ErrNoVerifiedEmail
	}
	return verified, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
