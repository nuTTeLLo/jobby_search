package response

import (
	"net/http"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

func Success(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

func SuccessMessage(message string) Response {
	return Response{
		Success: true,
		Message: message,
	}
}

func Error(message string) Response {
	return Response{
		Success: false,
		Error:   message,
	}
}

func WithStatusCode(w http.ResponseWriter, r Response, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// Write response - using encoder in handler
}
