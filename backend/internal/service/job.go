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
	CountryIndeed string `json:"country_indeed"`
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
	Source          string  `json:"source"`
}

func (s *JobService) CreateJob(userID string, input *domain.JobCreateInput) (*domain.Job, error) {
	exists, err := s.repo.ExistsByURL(input.JobURL, userID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, appErrors.ErrAlreadyExists
	}

	job := &domain.Job{
		UserID:      userID,
		JobTitle:    input.JobTitle,
		CompanyName: input.CompanyName,
		Location:    input.Location,
		JobURL:      input.JobURL,
		Description: input.Description,
		Salary:      input.Salary,
		JobType:     input.JobType,
		IsRemote:    input.IsRemote,
		EasyApply:   input.EasyApply,
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

func (s *JobService) GetJob(userID, id string) (*domain.Job, error) {
	return s.repo.GetByID(id, userID)
}

func (s *JobService) GetAllJobs(userID string, filter *domain.JobFilter) ([]domain.Job, error) {
	if filter == nil {
		filter = &domain.JobFilter{}
	}
	filter.UserID = userID
	return s.repo.GetAll(filter)
}

func (s *JobService) UpdateJob(userID, id string, input *domain.JobUpdateInput) (*domain.Job, error) {
	job, err := s.repo.GetByID(id, userID)
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
	job.EasyApply = input.EasyApply

	if err := s.repo.Update(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) UpdateJobStatus(userID, id string, status string) (*domain.Job, error) {
	job, err := s.repo.GetByID(id, userID)
	if err != nil {
		return nil, err
	}
	job.Status = status

	if err := s.repo.Update(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) DeleteJob(userID, id string) error {
	return s.repo.Delete(id, userID)
}

type SearchResult struct {
	domain.Job
	IsSaved bool `json:"is_saved"`
}

func (s *JobService) SearchJobs(userID string, params MCPSearchParams) ([]SearchResult, error) {
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
		jobURL := mcpJob.JobURL
		if jobURL == "" {
			jobURL = mcpJob.JobURLDirect
		}
		if jobURL == "" {
			jobURL = mcpJob.URL
		}

		if jobURL != "" {
			if seenURLs[jobURL] {
				continue
			}
			seenURLs[jobURL] = true
		}

		jobTitle := mcpJob.JobTitle
		if jobTitle == "" {
			jobTitle = mcpJob.Title
		}
		if jobTitle == "" {
			jobTitle = mcpJob.Summary
		}

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

		isSaved := false
		if jobURL != "" {
			exists, err := s.repo.ExistsByURL(jobURL, userID)
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
			Source:      mcpJob.Source,
			Status:      string(domain.StatusNew),
		}
		results = append(results, SearchResult{Job: job, IsSaved: isSaved})
	}

	return results, nil
}

// Attachment constants
const (
	MaxFileSize                int64 = 10 * 1024 * 1024 // 10MB
	AllowedFileTypeResume            = "resume"
	AllowedFileTypeCoverLetter       = "cover_letter"
)

var allowedMIMETypes = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

type AttachmentInput struct {
	JobID    string
	UserID   string
	FileName string
	FileType string // "resume" or "cover_letter"
	MIMEType string
	Data     []byte
}

func (s *JobService) CreateAttachment(input *AttachmentInput) (*domain.Attachment, error) {
	if input.FileType != AllowedFileTypeResume && input.FileType != AllowedFileTypeCoverLetter {
		return nil, fmt.Errorf("invalid file type: %s (must be 'resume' or 'cover_letter')", input.FileType)
	}

	if !allowedMIMETypes[input.MIMEType] {
		return nil, fmt.Errorf("invalid MIME type: %s (allowed: application/pdf, application/msword, application/vnd.openxmlformats-officedocument.wordprocessingml.document)", input.MIMEType)
	}

	if int64(len(input.Data)) > MaxFileSize {
		return nil, fmt.Errorf("file too large: max size is 10MB")
	}

	// Verify job exists and belongs to this user
	_, err := s.repo.GetByID(input.JobID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	attachment := &domain.Attachment{
		JobID:    input.JobID,
		FileName: input.FileName,
		FileType: input.FileType,
		MIMEType: input.MIMEType,
		Data:     input.Data,
		FileSize: int64(len(input.Data)),
	}

	if err := s.repo.CreateAttachment(attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

func (s *JobService) GetAttachment(id string) (*domain.Attachment, error) {
	return s.repo.GetAttachmentByID(id)
}

func (s *JobService) GetAttachmentsByJobID(jobID string) ([]domain.Attachment, error) {
	return s.repo.GetAttachmentsByJobID(jobID)
}

func (s *JobService) DeleteAttachment(id string) error {
	return s.repo.DeleteAttachment(id)
}
