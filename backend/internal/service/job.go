package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/repository"
	appErrors "job-tracker-backend/pkg/errors"
)

type JobService struct {
	repo         *repository.JobRepository
	mcpServerURL string
	httpClient   *http.Client
}

func NewJobService(repo *repository.JobRepository, mcpServerURL string) *JobService {
	return &JobService{
		repo:         repo,
		mcpServerURL: mcpServerURL,
		httpClient:   &http.Client{},
	}
}

type MCPSearchParams struct {
	SiteNames     string `json:"site_names"`
	SearchTerm    string `json:"search_term"`
	Location      string `json:"location"`
	Distance      int    `json:"distance"`
	JobType       string `json:"job_type"`
	ResultsWanted int    `json:"results_wanted"`
	HoursOld      int    `json:"hours_old"`
	IsRemote      bool   `json:"is_remote"`
	Format        string `json:"format"`
}

type MCPSearchRequest struct {
	Method string          `json:"method"`
	Params MCPSearchParams `json:"params"`
}

type MCPSearchResponse struct {
	Count   int      `json:"count"`
	Message string   `json:"message"`
	Jobs    []MCPJob `json:"jobs"`
}

type MCPJob struct {
	JobTitle        string  `json:"jobTitle"`
	JobSummary      string  `json:"jobSummary"`
	Description     string  `json:"description"`
	JobURL          string  `json:"jobUrl"`
	JobURLDirect    string  `json:"jobUrlDirect"`
	Location        string  `json:"location"`
	Country         string  `json:"country"`
	State           string  `json:"state"`
	City            string  `json:"city"`
	DatePosted      string  `json:"datePosted"`
	JobType         string  `json:"jobType"`
	Salary          string  `json:"salary"`
	SalaryPeriod    string  `json:"salaryPeriod"`
	MinAmount       float64 `json:"minAmount"`
	MaxAmount       float64 `json:"maxAmount"`
	IsRemote        bool    `json:"isRemote"`
	CompanyName     string  `json:"companyName"`
	CompanyIndustry string  `json:"companyIndustry"`
	CompanyURL      string  `json:"companyUrl"`
	CompanyLogo     string  `json:"companyLogo"`
	Title           string  `json:"title"`
	Summary         string  `json:"summary"`
	URL             string  `json:"url"`
	Company         string  `json:"company"`
}

func (s *JobService) CreateJob(input *domain.JobCreateInput) (*domain.Job, error) {
	exists, err := s.repo.ExistsByURL(input.JobURL)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, appErrors.ErrAlreadyExists
	}

	job := &domain.Job{
		JobTitle:    input.JobTitle,
		CompanyName: input.CompanyName,
		Location:    input.Location,
		JobURL:      input.JobURL,
		Description: input.Description,
		Salary:      input.Salary,
		JobType:     input.JobType,
		IsRemote:    input.IsRemote,
		Source:      input.Source,
		Status:      string(domain.StatusNew),
		Notes:       input.Notes,
	}

	if job.Source == "" {
		job.Source = "manual"
	}

	if err := s.repo.Create(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) GetJob(id string) (*domain.Job, error) {
	return s.repo.GetByID(id)
}

func (s *JobService) GetAllJobs(filter *domain.JobFilter) ([]domain.Job, error) {
	return s.repo.GetAll(filter)
}

func (s *JobService) UpdateJob(id string, input *domain.JobUpdateInput) (*domain.Job, error) {
	job, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if input.JobTitle != "" {
		job.JobTitle = input.JobTitle
	}
	if input.CompanyName != "" {
		job.CompanyName = input.CompanyName
	}
	if input.Location != "" {
		job.Location = input.Location
	}
	if input.JobURL != "" {
		job.JobURL = input.JobURL
	}
	if input.Description != "" {
		job.Description = input.Description
	}
	if input.Salary != "" {
		job.Salary = input.Salary
	}
	if input.JobType != "" {
		job.JobType = input.JobType
	}
	if input.Source != "" {
		job.Source = input.Source
	}
	if input.Status != "" {
		job.Status = input.Status
	}
	if input.Notes != "" {
		job.Notes = input.Notes
	}
	job.IsRemote = input.IsRemote

	if err := s.repo.Update(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) UpdateJobStatus(id string, status string) (*domain.Job, error) {
	job, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	job.Status = status

	if err := s.repo.Update(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) DeleteJob(id string) error {
	return s.repo.Delete(id)
}

type SearchResult struct {
	domain.Job
	IsSaved bool `json:"is_saved"`
}

func (s *JobService) SearchJobs(params MCPSearchParams) ([]SearchResult, error) {
	reqBody := MCPSearchRequest{
		Method: "search_jobs",
		Params: params,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	mcpURL := fmt.Sprintf("%s/api", s.mcpServerURL)
	resp, err := s.httpClient.Post(mcpURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read body first
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var mcpResp MCPSearchResponse
	if err := json.Unmarshal(bodyBytes, &mcpResp); err != nil {
		return nil, fmt.Errorf("failed to decode MCP response: %w", err)
	}

	var results []SearchResult
	seenURLs := make(map[string]bool)
	for _, mcpJob := range mcpResp.Jobs {
		// Check multiple field names for URL and title
		jobURL := mcpJob.JobURL
		if jobURL == "" {
			jobURL = mcpJob.JobURLDirect
		}
		if jobURL == "" {
			jobURL = mcpJob.URL
		}

		// Skip duplicate URLs only if URL is not empty
		if jobURL != "" {
			if seenURLs[jobURL] {
				continue
			}
			seenURLs[jobURL] = true
		}

		// Use title from various possible fields
		jobTitle := mcpJob.JobTitle
		if jobTitle == "" {
			jobTitle = mcpJob.Title
		}
		if jobTitle == "" {
			jobTitle = mcpJob.Summary
		}

		// Skip if no title
		if jobTitle == "" {
			continue
		}

		companyName := mcpJob.CompanyName
		if companyName == "" {
			companyName = mcpJob.Company
		}

		salary := mcpJob.Salary
		if salary == "" && (mcpJob.MinAmount > 0 || mcpJob.MaxAmount > 0) {
			salary = fmt.Sprintf("%.0f-%.0f", mcpJob.MinAmount, mcpJob.MaxAmount)
		}

		// Check if job already exists in database
		isSaved := false
		if jobURL != "" {
			exists, err := s.repo.ExistsByURL(jobURL)
			if err == nil && exists {
				isSaved = true
			}
		}

		job := domain.Job{
			JobTitle:    jobTitle,
			CompanyName: companyName,
			Location:    mcpJob.Location,
			JobURL:      jobURL,
			Description: mcpJob.Description,
			Salary:      salary,
			JobType:     mcpJob.JobType,
			IsRemote:    mcpJob.IsRemote,
			Source:      "mcp",
			Status:      string(domain.StatusNew),
		}
		results = append(results, SearchResult{Job: job, IsSaved: isSaved})
	}

	return results, nil
}
