package fit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"job-tracker-backend/internal/domain"

	"golang.org/x/net/html"
)

// ErrFetch means the posting could not be read from its URL. The caller can send
// the posting's fields instead.
var ErrFetch = errors.New("could not read the posting")

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	linkedInPostingURL = "https://www.linkedin.com/jobs-guest/jobs/api/jobPosting/"
	seekGraphQLURL     = "https://www.seek.com.au/graphql"
	// An employer's careers page carries menus and boilerplate around the posting;
	// past this many characters it is almost all noise.
	maxPageTextLength = 40000
	// Less text than this from a page means it renders the posting with
	// JavaScript, or blocked the request.
	minPageTextLength = 200
	maxPageBytes      = 5 << 20
)

var (
	// /jobs/view/4483536053 or /jobs/view/senior-engineer-at-acme-4483536053
	linkedInViewPath = regexp.MustCompile(`/jobs/view/(?:[^/]*-)?(\d{6,})`)
	seekJobPath      = regexp.MustCompile(`/job/(\d+)`)
	digitsOnly       = regexp.MustCompile(`^\d+$`)
)

type Fetcher struct {
	client *http.Client
}

func NewFetcher() *Fetcher {
	// The URL is the caller's, and this runs inside the cluster: refuse to connect
	// anywhere but the public internet, redirects included.
	dialer := &net.Dialer{Timeout: 10 * time.Second, Control: publicAddressesOnly}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext
	transport.Proxy = nil
	return &Fetcher{client: &http.Client{Timeout: 30 * time.Second, Transport: transport}}
}

// publicAddressesOnly runs after DNS resolution, so a public name that resolves to
// a private address is refused too.
func publicAddressesOnly(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() ||
		// 100.64.0.0/10: carrier-grade NAT, which Tailscale addresses fall in.
		(ip.To4() != nil && ip.To4()[0] == 100 && ip.To4()[1]&0xc0 == 64) {
		return fmt.Errorf("%s is not a public address", host)
	}
	return nil
}

// Fetch reads a posting from its URL. LinkedIn and Seek are read through the same
// public endpoints the board scrape uses, which give the posting without the
// sign-in wall; anything else is fetched as a page and reduced to its text.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*domain.FitPosting, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("%w: not an http(s) URL", ErrFetch)
	}

	var posting *domain.FitPosting
	switch host := strings.ToLower(parsed.Hostname()); {
	case strings.HasSuffix(host, "linkedin.com"):
		id := linkedInJobID(parsed)
		if id == "" {
			return nil, fmt.Errorf("%w: no job id in the LinkedIn URL", ErrFetch)
		}
		posting, err = f.fetchLinkedIn(ctx, id)
	case strings.Contains(host, "seek."):
		id := seekJobID(parsed)
		if id == "" {
			return nil, fmt.Errorf("%w: no job id in the Seek URL", ErrFetch)
		}
		posting, err = f.fetchSeek(ctx, id)
	default:
		posting, err = f.fetchPage(ctx, parsed.String())
	}
	if err != nil {
		return nil, err
	}
	posting.URL = parsed.String()
	return posting, nil
}

func linkedInJobID(u *url.URL) string {
	// Search and collection pages name the open posting in currentJobId.
	if id := u.Query().Get("currentJobId"); digitsOnly.MatchString(id) {
		return id
	}
	if m := linkedInViewPath.FindStringSubmatch(u.Path); m != nil {
		return m[1]
	}
	return ""
}

func seekJobID(u *url.URL) string {
	if id := u.Query().Get("jobId"); digitsOnly.MatchString(id) {
		return id
	}
	if m := seekJobPath.FindStringSubmatch(u.Path); m != nil {
		return m[1]
	}
	return ""
}

func (f *Fetcher) fetchLinkedIn(ctx context.Context, id string) (*domain.FitPosting, error) {
	body, err := f.get(ctx, linkedInPostingURL+id, nil)
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: unreadable LinkedIn page", ErrFetch)
	}
	posting := &domain.FitPosting{
		Source:      "linkedin",
		JobTitle:    textByClass(doc, "top-card-layout__title"),
		CompanyName: textByClass(doc, "topcard__org-name-link"),
		Location:    textByClass(doc, "topcard__flavor--bullet"),
	}
	if markup := findByClass(doc, "show-more-less-html__markup"); markup != nil {
		posting.Description = textOf(markup)
	}
	if posting.JobTitle == "" {
		return nil, fmt.Errorf("%w: LinkedIn has no posting %s (closed, or the guest page changed)", ErrFetch, id)
	}
	return posting, nil
}

func (f *Fetcher) fetchSeek(ctx context.Context, id string) (*domain.FitPosting, error) {
	query, _ := json.Marshal(map[string]interface{}{
		"operationName": "jobDetails",
		"variables":     map[string]string{"jobId": id},
		"query": "query jobDetails($jobId: ID!) { jobDetails(id: $jobId) { job { " +
			"title advertiser { name } location { label } content(platform: WEB) } } }",
	})
	body, err := f.post(ctx, seekGraphQLURL, query, map[string]string{
		"Content-Type":         "application/json",
		"seek-request-brand":   "seek",
		"seek-request-country": "AU",
	})
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Data struct {
			JobDetails *struct {
				Job struct {
					Title      string `json:"title"`
					Advertiser struct {
						Name string `json:"name"`
					} `json:"advertiser"`
					Location struct {
						Label string `json:"label"`
					} `json:"location"`
					Content string `json:"content"`
				} `json:"job"`
			} `json:"jobDetails"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Data.JobDetails == nil {
		return nil, fmt.Errorf("%w: Seek has no posting %s", ErrFetch, id)
	}
	job := parsed.Data.JobDetails.Job
	return &domain.FitPosting{
		Source:      "seek",
		JobTitle:    job.Title,
		CompanyName: job.Advertiser.Name,
		Location:    job.Location.Label,
		Description: htmlToText(job.Content),
	}, nil
}

// fetchPage reads any other posting page, typically an employer's careers site or
// ATS. The title, company and location are left for the model to find in the text.
func (f *Fetcher) fetchPage(ctx context.Context, pageURL string) (*domain.FitPosting, error) {
	body, err := f.get(ctx, pageURL, map[string]string{"Accept": "text/html,application/xhtml+xml"})
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: unreadable page", ErrFetch)
	}

	posting := &domain.FitPosting{Source: "web"}
	if title := findElement(doc, "title"); title != nil {
		posting.JobTitle = strings.Join(strings.Fields(textOf(title)), " ")
	}
	text := textOf(doc)
	if utf8.RuneCountInString(text) < minPageTextLength {
		return nil, fmt.Errorf("%w: the page has almost no text (it may need JavaScript); send the description instead", ErrFetch)
	}
	if runes := []rune(text); len(runes) > maxPageTextLength {
		text = string(runes[:maxPageTextLength]) + "\n[page text cut off here]"
	}
	posting.Description = text
	return posting, nil
}

func (f *Fetcher) get(ctx context.Context, target string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetch, err)
	}
	return f.do(req, headers)
}

func (f *Fetcher) post(ctx context.Context, target string, body []byte, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetch, err)
	}
	return f.do(req, headers)
}

func (f *Fetcher) do(req *http.Request, headers map[string]string) ([]byte, error) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "en-AU,en;q=0.9")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetch, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s answered HTTP %d", ErrFetch, req.URL.Host, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetch, err)
	}
	return body, nil
}
