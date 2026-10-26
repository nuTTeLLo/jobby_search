// Command fitsweep scores the tracker's unscored discovered postings against the
// user's fit profile, stores each score, and sends a Telegram "top fits" summary.
//
// It runs on the user's Mac from launchd at 09:30, after the 09:00 board scrape,
// and scores with the local Claude Code CLI on the user's Claude subscription.
// It runs there rather than in the cluster because the node has no memory to spare
// for the CLI; see AGENTS.md. Everything it needs comes from the environment:
//
//	TRACKER_URL                       tracker base URL (default the tailnet one)
//	TRACKER_USER_ID, TRACKER_EMAIL    the account to mint a token for
//	JWT_SECRET                        the tracker's signing secret
//	TELEGRAM_BOT_TOKEN, TELEGRAM_CHAT_ID, DISCOVERED_URL   optional summary
//	FIT_CLAUDE_BIN, FIT_MODEL, FIT_EFFORT, FIT_MAX_PER_RUN  optional overrides
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/fit"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultTrackerURL = "https://job-tracker.tailee323f.ts.net"
	// Past this many postings in one run, something upstream is wrong (a broken
	// title filter), and scoring them all would eat the plan's usage limits.
	defaultMaxPerRun = 100
	// After this many failures in a row it is the setup, not the postings.
	maxConsecutiveFailures = 3
	// The whole run, so a stuck CLI cannot hold the launchd job forever.
	runTimeout = 45 * time.Minute
	// Postings scoring at least this make the Telegram summary.
	topFitMinScore = 4
	topFitMax      = 10
)

type config struct {
	trackerURL    string
	userID        string
	email         string
	jwtSecret     string
	telegramToken string
	telegramChat  string
	discoveredURL string
	claudeBin     string
	model         string
	effort        string
	maxPerRun     int
}

func loadConfig() (*config, error) {
	cfg := &config{
		trackerURL:    strings.TrimRight(envOr("TRACKER_URL", defaultTrackerURL), "/"),
		userID:        os.Getenv("TRACKER_USER_ID"),
		email:         os.Getenv("TRACKER_EMAIL"),
		jwtSecret:     os.Getenv("JWT_SECRET"),
		telegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		telegramChat:  os.Getenv("TELEGRAM_CHAT_ID"),
		discoveredURL: os.Getenv("DISCOVERED_URL"),
		claudeBin:     envOr("FIT_CLAUDE_BIN", "claude"),
		model:         envOr("FIT_MODEL", "claude-opus-5-5"),
		effort:        envOr("FIT_EFFORT", "low"),
		maxPerRun:     defaultMaxPerRun,
	}
	if value := os.Getenv("FIT_MAX_PER_RUN"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return nil, fmt.Errorf("FIT_MAX_PER_RUN must be a positive number, got %q", value)
		}
		cfg.maxPerRun = n
	}
	var missing []string
	for name, value := range map[string]string{
		"TRACKER_USER_ID": cfg.userID, "TRACKER_EMAIL": cfg.email, "JWT_SECRET": cfg.jwtSecret,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	dryRun := flag.Bool("dry-run", false, "score and print, but store nothing and send nothing")
	flag.Parse()
	log.SetFlags(log.LstdFlags)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("fitsweep: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	if err := run(ctx, cfg, *dryRun); err != nil {
		log.Fatalf("fitsweep: %v", err)
	}
}

// scoredRole is one role (one match_key) and its fit.
type scoredRole struct {
	job domain.UnscoredDiscoveredJob
	ids []string
	fit *domain.Fit
}

func run(ctx context.Context, cfg *config, dryRun bool) error {
	tracker, err := newTrackerClient(cfg)
	if err != nil {
		return err
	}

	var profile domain.Profile
	if err := tracker.do(ctx, http.MethodGet, "/api/profile", nil, &profile); err != nil {
		return fmt.Errorf("load profile: %w", err)
	}
	if strings.TrimSpace(profile.FitProfile) == "" {
		log.Print("no fit profile saved on the Profile page; nothing to score against")
		return nil
	}

	var unscored []domain.UnscoredDiscoveredJob
	if err := tracker.do(ctx, http.MethodGet, "/api/discovered-jobs/unscored", nil, &unscored); err != nil {
		return fmt.Errorf("load unscored postings: %w", err)
	}
	roles := groupByRole(unscored)
	log.Printf("%d unscored posting(s), %d role(s)", len(unscored), len(roles))
	if len(roles) == 0 {
		return nil
	}

	scorer := fit.NewScorer(cfg.claudeBin, cfg.model, cfg.effort)
	var scored []scoredRole
	var warnings []string
	failuresInARow := 0

	for _, role := range roles {
		if len(scored) >= cfg.maxPerRun {
			warnings = append(warnings, fmt.Sprintf("stopped at FIT_MAX_PER_RUN=%d; the rest wait for the next run", cfg.maxPerRun))
			break
		}

		result, err := scorer.Score(ctx, profile.FitProfile, &domain.FitPosting{
			Source:      role.job.Source,
			JobTitle:    role.job.JobTitle,
			CompanyName: role.job.CompanyName,
			Location:    role.job.Location,
			Description: role.job.Description,
		})
		if err == nil && !dryRun {
			for _, id := range role.ids {
				if putErr := tracker.do(ctx, http.MethodPut, "/api/discovered-jobs/"+url.PathEscape(id)+"/fit", result, nil); putErr != nil {
					err = fmt.Errorf("store fit: %w", putErr)
					break
				}
			}
		}
		if err != nil {
			label := fmt.Sprintf("%s - %s", role.job.CompanyName, role.job.JobTitle)
			log.Printf("FAILED %s: %v", label, err)
			warnings = append(warnings, fmt.Sprintf("%s: %v", label, err))
			failuresInARow++
			if errors.Is(err, fit.ErrUnavailable) || ctx.Err() != nil || failuresInARow >= maxConsecutiveFailures {
				warnings = append(warnings, "scoring stopped early; the rest wait for the next run")
				break
			}
			continue
		}

		failuresInARow = 0
		role.fit = result
		scored = append(scored, role)
		log.Printf("fit %d  %s - %s", result.Score, role.job.CompanyName, role.job.JobTitle)
	}

	log.Printf("scored %d role(s), %d warning(s)", len(scored), len(warnings))
	message := formatTelegram(scored, warnings, time.Now(), cfg.discoveredURL)
	if dryRun {
		fmt.Println("\nDry run: nothing stored, Telegram not sent. It would say:\n\n" + message)
		return nil
	}
	if message != "" {
		sendTelegram(ctx, cfg, message)
	}
	return nil
}

// groupByRole folds postings of the same role on several boards into one, in the
// order the tracker sent them.
func groupByRole(jobs []domain.UnscoredDiscoveredJob) []scoredRole {
	var roles []scoredRole
	index := map[string]int{}
	for _, job := range jobs {
		if i, ok := index[job.MatchKey]; ok {
			roles[i].ids = append(roles[i].ids, job.ID)
			// Score the copy with the most to read.
			if len(job.Description) > len(roles[i].job.Description) {
				roles[i].job = job
			}
			continue
		}
		index[job.MatchKey] = len(roles)
		roles = append(roles, scoredRole{job: job, ids: []string{job.ID}})
	}
	return roles
}

var fitMarkers = map[int]string{5: "🟢", 4: "🟢", 3: "🟡", 2: "⚪", 1: "⚪"}

// formatTelegram is the "top fits today" message, or "" when there is nothing to
// say (no postings were waiting).
func formatTelegram(scored []scoredRole, warnings []string, now time.Time, discoveredURL string) string {
	if len(scored) == 0 && len(warnings) == 0 {
		return ""
	}

	var good, possible, weak int
	for _, role := range scored {
		switch {
		case role.fit.Score >= topFitMinScore:
			good++
		case role.fit.Score == 3:
			possible++
		default:
			weak++
		}
	}

	top := make([]scoredRole, 0, len(scored))
	for _, role := range scored {
		if role.fit.Score >= topFitMinScore {
			top = append(top, role)
		}
	}
	sort.SliceStable(top, func(i, j int) bool { return top[i].fit.Score > top[j].fit.Score })

	var b strings.Builder
	fmt.Fprintf(&b, "⭐ Top fits — %s\n\n", now.Format("2 Jan"))
	fmt.Fprintf(&b, "Scored %d role%s: %d good or better, %d possible, %d weak.\n",
		len(scored), plural(len(scored)), good, possible, weak)

	if len(top) == 0 && len(scored) > 0 {
		b.WriteString("\nNo strong fits today.\n")
	}
	for i, role := range top {
		if i == topFitMax {
			fmt.Fprintf(&b, "\n…and %d more\n", len(top)-topFitMax)
			break
		}
		company := role.job.CompanyName
		if company == "" {
			company = "?"
		}
		fmt.Fprintf(&b, "\n%s %d/5 %s — %s\n", fitMarkers[role.fit.Score], role.fit.Score, company, role.job.JobTitle)
		if role.fit.Summary != "" {
			fmt.Fprintf(&b, "%s\n", role.fit.Summary)
		}
		if len(role.fit.Flags) > 0 {
			fmt.Fprintf(&b, "⚠ %s\n", strings.Join(role.fit.Flags, "; "))
		}
		if role.job.JobURL != "" {
			fmt.Fprintf(&b, "%s\n", role.job.JobURL)
		}
	}

	if len(warnings) > 0 {
		b.WriteString("\n⚠️ Warnings:\n")
		for _, warning := range warnings {
			fmt.Fprintf(&b, "• %s\n", warning)
		}
	}
	if discoveredURL != "" {
		fmt.Fprintf(&b, "\n%s\n", discoveredURL)
	}

	// Telegram caps a message at 4096 characters.
	text := strings.TrimSpace(b.String())
	if runes := []rune(text); len(runes) > 4000 {
		text = string(runes[:4000]) + "\n…"
	}
	return text
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// sendTelegram is best effort: a missing or failing bot never fails the sweep.
func sendTelegram(ctx context.Context, cfg *config, text string) {
	if cfg.telegramToken == "" || cfg.telegramChat == "" {
		log.Print("Telegram not configured; summary not sent")
		return
	}
	form := url.Values{
		"chat_id":                  {cfg.telegramChat},
		"text":                     {text},
		"disable_web_page_preview": {"true"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.telegram.org/bot"+cfg.telegramToken+"/sendMessage", strings.NewReader(form.Encode()))
	if err != nil {
		log.Printf("Telegram: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// The URL holds the bot token, so report only that the request failed.
		log.Print("Telegram: send failed (request error)")
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Telegram: send failed (HTTP %d)", resp.StatusCode)
		return
	}
	log.Print("Telegram: summary sent")
}

// trackerClient calls the tracker API as the configured user.
type trackerClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func newTrackerClient(cfg *config) (*trackerClient, error) {
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": cfg.userID,
		"email":   cfg.email,
		"iat":     now.Unix(),
		"exp":     now.Add(2 * time.Hour).Unix(),
	}).SignedString([]byte(cfg.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("mint tracker token: %w", err)
	}
	return &trackerClient{baseURL: cfg.trackerURL, token: token, http: &http.Client{Timeout: 30 * time.Second}}, nil
}

// do sends body (if any) as JSON and decodes the response's data into out (if
// any). Errors carry the tracker's own message.
func (c *trackerClient) do(ctx context.Context, method, path string, body, out interface{}) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   string          `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("HTTP %d, unreadable response", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || !envelope.Success {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, envelope.Error)
	}
	if out != nil {
		return json.Unmarshal(envelope.Data, out)
	}
	return nil
}
