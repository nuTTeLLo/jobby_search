package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"job-tracker-backend/internal/domain"
	appMiddleware "job-tracker-backend/internal/middleware"
	"job-tracker-backend/internal/service"
	appErrors "job-tracker-backend/pkg/errors"
	"job-tracker-backend/pkg/response"

	"github.com/go-chi/chi/v5"
)

type JobHandler struct {
	service *service.JobService
}

func NewJobHandler(svc *service.JobService) *JobHandler {
	return &JobHandler{service: svc}
}

type AttachmentHandler struct {
	service *service.JobService
}

func NewAttachmentHandler(svc *service.JobService) *AttachmentHandler {
	return &AttachmentHandler{service: svc}
}

func (h *JobHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.ListJobs)
	r.Post("/", h.CreateJob)
	r.Post("/search", h.SearchJobs)
	r.Get("/{id}", h.GetJob)
	r.Put("/{id}", h.UpdateJob)
	r.Delete("/{id}", h.DeleteJob)
	r.Patch("/{id}/status", h.UpdateJobStatus)

	attachmentHandler := NewAttachmentHandler(h.service)
	r.Mount("/api/jobs/{id}/attachments", attachmentHandler.Routes())

	return r
}

func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())

	filter := &domain.JobFilter{
		Status: r.URL.Query().Get("status"),
		Source: r.URL.Query().Get("source"),
	}

	jobs, err := h.service.GetAllJobs(userID, filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	if jobs == nil {
		jobs = []domain.Job{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(jobs))
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
	jobID := chi.URLParam(r, "id")

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

	attachment.Data = nil
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(attachment))
}

func (h *AttachmentHandler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", attachment.MIMEType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.FileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", attachment.FileSize))
	w.Write(attachment.Data)
}

func (h *AttachmentHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.service.DeleteAttachment(id)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.SuccessMessage("Attachment deleted successfully"))
}
