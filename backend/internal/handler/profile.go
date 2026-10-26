package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"job-tracker-backend/internal/domain"
	appMiddleware "job-tracker-backend/internal/middleware"
	"job-tracker-backend/internal/service"
	appErrors "job-tracker-backend/pkg/errors"
	"job-tracker-backend/pkg/response"

	"github.com/go-chi/chi/v5"
)

type ProfileHandler struct {
	service *service.ProfileService
}

func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{service: svc}
}

func (h *ProfileHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.GetProfile)
	r.Put("/", h.UpdateProfile)

	return r
}

func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())

	profile, err := h.service.Get(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(profile))
}

func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())

	var input domain.ProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		return
	}

	profile, err := h.service.Update(userID, &input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if err == appErrors.ErrInvalidInput {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response.Error(
				fmt.Sprintf("Fit profile must be at most %d characters", domain.MaxFitProfileLength),
			))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response.Error(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(profile))
}
