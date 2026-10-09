package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"job-tracker-backend/internal/auth"
	appMiddleware "job-tracker-backend/internal/middleware"
	"job-tracker-backend/internal/service"
	"job-tracker-backend/pkg/response"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

// oauthCookie carries the state and PKCE verifier from /start to /callback.
// It is scoped to /api/auth/ and lives only as long as a sign-in takes.
const (
	oauthCookie    = "oauth_flow"
	oauthCookieTTL = 10 * time.Minute
)

type AuthHandler struct {
	service     *service.AuthService
	frontendURL string
	// secureCookie is false only for plain-http local dev, where a Secure
	// cookie would never be sent back.
	secureCookie bool
}

func NewAuthHandler(svc *service.AuthService, frontendURL string, secureCookie bool) *AuthHandler {
	return &AuthHandler{service: svc, frontendURL: strings.TrimRight(frontendURL, "/"), secureCookie: secureCookie}
}

func (h *AuthHandler) PublicRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/{provider}/start", h.Start)
	r.Get("/{provider}/callback", h.Callback)
	return r
}

// Start sends the browser to the provider's sign-in page. The random state is
// remembered in a cookie so Callback can tell this browser started the flow —
// otherwise an attacker could finish a sign-in they began, logging the victim
// into the attacker's account.
func (h *AuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	state := oauth2.GenerateVerifier() // any long random string works as state
	verifier := oauth2.GenerateVerifier()

	authURL, err := h.service.AuthURL(chi.URLParam(r, "provider"), state, verifier)
	if err != nil {
		h.failLogin(w, r, "unknown_provider")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthCookie,
		Value:    state + "." + verifier,
		Path:     "/api/auth/",
		MaxAge:   int(oauthCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		// Lax, not Strict: the provider's redirect back is a cross-site
		// navigation, and Strict would drop the cookie on exactly that request.
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback is where the provider sends the browser back, with a one-time code
// and the state from Start.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	cookie, cookieErr := r.Cookie(oauthCookie)
	// Single use: clear it whatever happens next.
	http.SetCookie(w, &http.Cookie{Name: oauthCookie, Path: "/api/auth/", MaxAge: -1, HttpOnly: true, Secure: h.secureCookie})

	q := r.URL.Query()
	if q.Get("error") != "" { // e.g. the user clicked "Cancel" on the consent screen
		h.failLogin(w, r, "cancelled")
		return
	}
	if cookieErr != nil {
		h.failLogin(w, r, "expired")
		return
	}
	state, verifier, ok := strings.Cut(cookie.Value, ".")
	if !ok || q.Get("state") == "" || q.Get("state") != state {
		h.failLogin(w, r, "state_mismatch")
		return
	}

	resp, err := h.service.CompleteLogin(r.Context(), chi.URLParam(r, "provider"), q.Get("code"), verifier)
	switch {
	case errors.Is(err, service.ErrEmailNotAllowed):
		h.failLogin(w, r, "not_allowed")
		return
	case errors.Is(err, auth.ErrNoVerifiedEmail):
		h.failLogin(w, r, "no_verified_email")
		return
	case err != nil:
		log.Printf("oauth callback: %v", err)
		h.failLogin(w, r, "failed")
		return
	}

	// The JWT rides in the fragment (#...), which browsers never send to a
	// server, so it stays out of access logs and Referer headers.
	http.Redirect(w, r, h.frontendURL+"/auth/callback#token="+url.QueryEscape(resp.Token), http.StatusFound)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Me(appMiddleware.UserIDFromContext(r.Context()))
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response.Error("user not found"))
		return
	}
	json.NewEncoder(w).Encode(response.Success(user))
}

func (h *AuthHandler) failLogin(w http.ResponseWriter, r *http.Request, reason string) {
	http.Redirect(w, r, h.frontendURL+"/login?error="+url.QueryEscape(reason), http.StatusFound)
}
