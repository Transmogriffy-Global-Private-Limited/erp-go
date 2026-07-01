package httpx

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ErrorBody struct {
	Error ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if payload == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, status int, code string, message string) {
	JSON(w, status, ErrorBody{
		Error: ErrorPayload{
			Code:    code,
			Message: message,
		},
	})
}

func MethodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	Error(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}
