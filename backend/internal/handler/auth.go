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

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

func (h *AuthHandler) PublicRoutes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	return r
}


func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input domain.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("invalid request body"))
		return
	}

	resp, err := h.service.Register(&input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		switch err {
		case appErrors.ErrAlreadyExists:
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(response.Error("email already registered"))
		case appErrors.ErrInvalidInput:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response.Error("invalid email or password (minimum 8 characters)"))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response.Error("registration failed"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response.Success(resp))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input domain.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("invalid request body"))
		return
	}

	resp, err := h.service.Login(&input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response.Error("invalid email or password"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.Success(resp))
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := appMiddleware.UserIDFromContext(r.Context())

	var input domain.ChangePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response.Error("invalid request body"))
		return
	}

	if err := h.service.ChangePassword(userID, &input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		switch err {
		case appErrors.ErrInvalidInput:
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response.Error("current password is incorrect or new password is too short (minimum 8 characters)"))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response.Error("failed to change password"))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response.SuccessMessage("password changed successfully"))
}
