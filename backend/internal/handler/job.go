package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"job-tracker-backend/internal/domain"
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
	filter := &domain.JobFilter{
		Status: r.URL.Query().Get("status"),
		Source: r.URL.Query().Get("source"),
	}

	jobs, err := h.service.GetAllJobs(filter)
	if err != nil {
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if jobs == nil {
		jobs = []domain.Job{}
	}

	json.NewEncoder(w).Encode(response.Success(jobs))
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var input domain.JobCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	job, err := h.service.CreateJob(&input)
	if err != nil {
		if err == appErrors.ErrAlreadyExists {
			json.NewEncoder(w).Encode(response.Error("Job with this URL already exists"))
			w.WriteHeader(http.StatusConflict)
			return
		}
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response.Success(job))
}

func (h *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	job, err := h.service.GetJob(id)
	if err != nil {
		if err == appErrors.ErrNotFound {
			json.NewEncoder(w).Encode(response.Error("Job not found"))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response.Success(job))
}

func (h *JobHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input domain.JobUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	job, err := h.service.UpdateJob(id, &input)
	if err != nil {
		if err == appErrors.ErrNotFound {
			json.NewEncoder(w).Encode(response.Error("Job not found"))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response.Success(job))
}

func (h *JobHandler) UpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var input domain.JobStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	job, err := h.service.UpdateJobStatus(id, input.Status)
	if err != nil {
		if err == appErrors.ErrNotFound {
			json.NewEncoder(w).Encode(response.Error("Job not found"))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response.Success(job))
}

func (h *JobHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.service.DeleteJob(id)
	if err != nil {
		if err == appErrors.ErrNotFound {
			json.NewEncoder(w).Encode(response.Error("Job not found"))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response.SuccessMessage("Job deleted successfully"))
}

func (h *JobHandler) SearchJobs(w http.ResponseWriter, r *http.Request) {
	var params service.MCPSearchParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		w.WriteHeader(http.StatusBadRequest)
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

	jobs, err := h.service.SearchJobs(params)
	if err != nil {
		json.NewEncoder(w).Encode(response.Error("Failed to search jobs: " + err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

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
	jobID := chi.URLParam(r, "id")

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		json.NewEncoder(w).Encode(response.Error("Failed to parse form data"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		json.NewEncoder(w).Encode(response.Error("Failed to get file from form"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileType := r.FormValue("file_type")
	if fileType == "" {
		fileType = "resume"
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		json.NewEncoder(w).Encode(response.Error("Failed to read file"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	input := &service.AttachmentInput{
		JobID:    jobID,
		FileName: header.Filename,
		FileType: fileType,
		MIMEType: header.Header.Get("Content-Type"),
		Data:     fileBytes,
	}

	attachment, err := h.service.CreateAttachment(input)
	if err != nil {
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Return without the binary data
	attachment.Data = nil
	json.NewEncoder(w).Encode(response.Success(attachment))
}

func (h *AttachmentHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")

	attachments, err := h.service.GetAttachmentsByJobID(jobID)
	if err != nil {
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Don't send binary data in list
	for i := range attachments {
		attachments[i].Data = nil
	}

	if attachments == nil {
		attachments = []domain.Attachment{}
	}

	json.NewEncoder(w).Encode(response.Success(attachments))
}

func (h *AttachmentHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	attachment, err := h.service.GetAttachment(id)
	if err != nil {
		if err == appErrors.ErrNotFound {
			json.NewEncoder(w).Encode(response.Error("Attachment not found"))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Don't send binary data in get
	attachment.Data = nil
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
		if err == appErrors.ErrNotFound {
			json.NewEncoder(w).Encode(response.Error("Attachment not found"))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response.SuccessMessage("Attachment deleted successfully"))
}
