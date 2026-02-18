package handler

import (
	"encoding/json"
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

func (h *JobHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.ListJobs)
	r.Post("/", h.CreateJob)
	r.Post("/search", h.SearchJobs)
	r.Get("/{id}", h.GetJob)
	r.Put("/{id}", h.UpdateJob)
	r.Delete("/{id}", h.DeleteJob)
	r.Patch("/{id}/status", h.UpdateJobStatus)

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
