package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"job-tracker-backend/internal/domain"
	"job-tracker-backend/internal/fit"
	appMiddleware "job-tracker-backend/internal/middleware"
	"job-tracker-backend/internal/service"
	appErrors "job-tracker-backend/pkg/errors"
	"job-tracker-backend/pkg/response"
)

// A posting's description is rarely more than a few tens of kilobytes.
const maxFitCheckBody = 1 << 20

type FitHandler struct {
	service *service.FitService
}

func NewFitHandler(svc *service.FitService) *FitHandler {
	return &FitHandler{service: svc}
}

// CheckFit scores one posting against the caller's fit profile. It takes as long
// as the model does, typically several seconds.
func (h *FitHandler) CheckFit(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")

	var input domain.FitCheckInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxFitCheckBody)).Decode(&input); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("Invalid request body"))
		return
	}

	result, err := h.service.Check(r.Context(), userID, &input)
	if err != nil {
		status, message := http.StatusBadGateway, "Fit check failed: "+err.Error()
		switch {
		case errors.Is(err, appErrors.ErrInvalidInput):
			status, message = http.StatusBadRequest, "Give a url, or at least a job_title or description"
		case errors.Is(err, service.ErrFitProfileTooLong):
			status, message = http.StatusBadRequest, fmt.Sprintf("Fit profile must be at most %d characters", domain.MaxFitProfileLength)
		case errors.Is(err, service.ErrNoFitProfile):
			status, message = http.StatusBadRequest, "No fit profile saved; save one on the Profile page or send fit_profile"
		case errors.Is(err, fit.ErrFetch):
			status, message = http.StatusUnprocessableEntity, err.Error()
		case errors.Is(err, fit.ErrUnavailable):
			status, message = http.StatusServiceUnavailable, err.Error()
		default:
			log.Printf("fit check failed: %v", err)
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response.Error(message))
		return
	}

	json.NewEncoder(w).Encode(response.Success(result))
}
