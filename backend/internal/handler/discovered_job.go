package handler

import (
	"encoding/json"
	"net/http"

	"job-tracker-backend/internal/domain"
	appMiddleware "job-tracker-backend/internal/middleware"
	"job-tracker-backend/internal/service"
	appErrors "job-tracker-backend/pkg/errors"
	"job-tracker-backend/pkg/response"

	"github.com/go-chi/chi/v5"
)

type DiscoveredJobHandler struct {
	service *service.DiscoveredJobService
}

func NewDiscoveredJobHandler(svc *service.DiscoveredJobService) *DiscoveredJobHandler {
	return &DiscoveredJobHandler{service: svc}
}

func (h *DiscoveredJobHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.ListDiscoveredJobs)
	r.Post("/", h.IngestDiscoveredJobs)
	r.Patch("/{id}/dismiss", h.DismissDiscoveredJob)

	return r
}

func (h *DiscoveredJobHandler) ListDiscoveredJobs(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	includeDismissed := r.URL.Query().Get("include_dismissed") == "true"

	jobs, err := h.service.GetRecent(userID, includeDismissed)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	if jobs == nil {
		jobs = []domain.DiscoveredJob{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(jobs))
}

func (h *DiscoveredJobHandler) IngestDiscoveredJobs(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())

	var input domain.DiscoveredIngestInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		return
	}

	result, err := h.service.Ingest(userID, &input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrInvalidInput {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response.Error("Each job requires external_id and job_title"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(result))
}

func (h *DiscoveredJobHandler) DismissDiscoveredJob(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	job, err := h.service.Dismiss(userID, id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response.Error("Discovered job not found"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(job))
}
