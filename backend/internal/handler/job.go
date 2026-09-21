package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"job-tracker-backend/internal/auth"
	"job-tracker-backend/internal/domain"
	appMiddleware "job-tracker-backend/internal/middleware"
	"job-tracker-backend/internal/service"
	appErrors "job-tracker-backend/pkg/errors"
	"job-tracker-backend/pkg/response"

	"github.com/go-chi/chi/v5"
)

type JobHandler struct {
	service   *service.JobService
	jwtSecret string
}

func NewJobHandler(svc *service.JobService, jwtSecret string) *JobHandler {
	return &JobHandler{service: svc, jwtSecret: jwtSecret}
}

type AttachmentHandler struct {
	service *service.JobService
	// jwtSecret signs the short-lived tokens that let a browser tab fetch one
	// attachment without an Authorization header.
	jwtSecret string
}

func NewAttachmentHandler(svc *service.JobService, jwtSecret string) *AttachmentHandler {
	return &AttachmentHandler{service: svc, jwtSecret: jwtSecret}
}

// ViewTokenTTL is deliberately short: the token ends up in a URL, so it should
// outlive the click that made it by little more than the fetch itself.
const ViewTokenTTL = 2 * time.Minute

func (h *JobHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.ListJobs)
	r.Post("/", h.CreateJob)
	r.Post("/search", h.SearchJobs)
	r.Get("/{id}", h.GetJob)
	r.Put("/{id}", h.UpdateJob)
	r.Delete("/{id}", h.DeleteJob)
	r.Patch("/{id}/status", h.UpdateJobStatus)

	attachmentHandler := NewAttachmentHandler(h.service, h.jwtSecret)
	r.Mount("/api/jobs/{id}/attachments", attachmentHandler.Routes())

	return r
}

func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())

	query := r.URL.Query()
	filter := &domain.JobFilter{
		Status: query.Get("status"),
		Source: query.Get("source"),
		Search: query.Get("q"),
		Sort:   query.Get("sort"),
		Order:  query.Get("order"),
	}

	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("page_size"))

	result, err := h.service.GetAllJobs(userID, filter, page, pageSize)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(result))
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())

	var input domain.JobCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		return
	}

	job, err := h.service.CreateJob(userID, &input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrAlreadyExists {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(response.Error("Job with this URL already exists"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(job))
}

func (h *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	job, err := h.service.GetJob(userID, id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response.Error("Job not found"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(job))
}

func (h *JobHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var input domain.JobUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		return
	}

	job, err := h.service.UpdateJob(userID, id, &input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response.Error("Job not found"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(job))
}

func (h *JobHandler) UpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var input domain.JobStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		return
	}

	job, err := h.service.UpdateJobStatus(userID, id, input.Status)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response.Error("Job not found"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(job))
}

func (h *JobHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	err := h.service.DeleteJob(userID, id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response.Error("Job not found"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.SuccessMessage("Job deleted successfully"))
}

func (h *JobHandler) SearchJobs(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())

	var params service.MCPSearchParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		return
	}

	if params.ResultsWanted == 0 {
		params.ResultsWanted = 20
	}
	if params.Distance == 0 {
		params.Distance = 50
	}
	if params.HoursOld == 0 {
		params.HoursOld = 72
	}
	if params.Format == "" {
		params.Format = "json"
	}

	jobs, err := h.service.SearchJobs(userID, params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error("Failed to search jobs: " + err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(map[string]interface{}{
		"count": len(jobs),
		"jobs":  jobs,
	}))
}

func (h *AttachmentHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/", h.UploadAttachment)
	r.Get("/", h.ListAttachments)
	r.Get("/{id}", h.GetAttachment)
	r.Get("/{id}/download", h.DownloadAttachment)
	r.Post("/{id}/view-token", h.CreateViewToken)
	r.Delete("/{id}", h.DeleteAttachment)

	return r
}

func (h *AttachmentHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	jobID := chi.URLParam(r, "id")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Failed to parse form data"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Failed to get file from form"))
		return
	}
	defer file.Close()

	fileType := r.FormValue("file_type")
	if fileType == "" {
		fileType = "resume"
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error("Failed to read file"))
		return
	}

	input := &service.AttachmentInput{
		JobID:    jobID,
		UserID:   userID,
		FileName: header.Filename,
		FileType: fileType,
		MIMEType: header.Header.Get("Content-Type"),
		Data:     fileBytes,
	}

	attachment, err := h.service.CreateAttachment(input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	attachment.Data = nil
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(attachment))
}

func (h *AttachmentHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	jobID := chi.URLParam(r, "id")

	if _, err := h.service.GetJob(userID, jobID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response.Error("Job not found"))
		return
	}

	attachments, err := h.service.GetAttachmentsByJobID(jobID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	for i := range attachments {
		attachments[i].Data = nil
	}

	if attachments == nil {
		attachments = []domain.Attachment{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(attachments))
}

func (h *AttachmentHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	attachment, err := h.service.GetAttachment(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response.Error("Attachment not found"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	if _, err := h.service.GetJob(userID, attachment.JobID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response.Error("Attachment not found"))
		return
	}

	attachment.Data = nil
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(attachment))
}

func (h *AttachmentHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	attachment, err := h.service.GetAttachment(id)
	if err != nil {
		if err == appErrors.ErrNotFound {
			http.Error(w, "Attachment not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := h.service.GetJob(userID, attachment.JobID); err != nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	// Always a download, never rendered: HTML attachments (interview prep pages)
	// must not execute on the tracker's origin. Viewing goes through
	// ViewAttachment, which isolates the page instead.
	w.Header().Set("Content-Type", attachment.MIMEType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.FileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", attachment.FileSize))
	w.Write(attachment.Data)
}

// CreateViewToken mints a short-lived token for one attachment, so the client
// can open it as an ordinary URL in a new tab.
func (h *AttachmentHandler) CreateViewToken(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	attachment, err := h.service.GetAttachment(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response.Error("Attachment not found"))
		return
	}
	// Ownership is checked here, at mint time, so the view route only has to
	// trust its own token.
	if _, err := h.service.GetJob(userID, attachment.JobID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response.Error("Attachment not found"))
		return
	}

	token, err := auth.GenerateViewToken(userID, id, h.jwtSecret, ViewTokenTTL)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error("Failed to create view token"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(map[string]interface{}{
		"token":      token,
		"url":        fmt.Sprintf("/api/attachments/%s/view?t=%s", id, url.QueryEscape(token)),
		"expires_in": int(ViewTokenTTL.Seconds()),
	}))
}

// inlineMIMETypes are the types a browser renders on its own. Everything else
// is sent as a download even through the view route, since an unrenderable
// inline response just produces an empty tab.
var inlineMIMETypes = map[string]bool{
	"text/html":       true,
	"application/pdf": true,
	"text/plain":      true,
	"text/markdown":   true,
}

// ViewAttachment serves one attachment for display in a browser tab. It is
// mounted outside the authenticated group: a tab navigation carries no
// Authorization header, so the caller presents a view token instead.
func (h *AttachmentHandler) ViewAttachment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	claims, err := auth.ValidateToken(r.URL.Query().Get("t"), h.jwtSecret)
	if err != nil || claims.Purpose != auth.PurposeAttachmentView || claims.AttachmentID != id {
		http.Error(w, "Invalid or expired view link", http.StatusUnauthorized)
		return
	}

	attachment, err := h.service.GetAttachment(id)
	if err != nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}
	if _, err := h.service.GetJob(claims.UserID, attachment.JobID); err != nil {
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	disposition := "attachment"
	if inlineMIMETypes[attachment.MIMEType] {
		disposition = "inline"
	}
	// Markdown has no browser renderer, so show the source rather than
	// prompting a save.
	contentType := attachment.MIMEType
	if contentType == "text/markdown" {
		contentType = "text/plain"
	}
	// Everything textual is stored as UTF-8, and it has to be declared: with no
	// charset a browser falls back to a legacy encoding (WebKit picks
	// Windows-1252), turning every em dash and curly quote into mojibake. The
	// header wins over any meta tag in the document, so a page that forgot its
	// own <meta charset> still renders correctly.
	if strings.HasPrefix(contentType, "text/") {
		contentType += "; charset=utf-8"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// An uploaded page is rendered on this origin, so sandbox it: without
	// allow-same-origin the document lands on an opaque origin and its scripts
	// cannot reach the tracker's own storage or session token.
	w.Header().Set("Content-Security-Policy", "sandbox allow-scripts allow-popups allow-forms")
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, attachment.FileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", attachment.FileSize))
	w.Write(attachment.Data)
}

func (h *AttachmentHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	attachment, err := h.service.GetAttachment(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response.Error("Attachment not found"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	if _, err := h.service.GetJob(userID, attachment.JobID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response.Error("Attachment not found"))
		return
	}

	if err := h.service.DeleteAttachment(id); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.SuccessMessage("Attachment deleted successfully"))
}
