package handlers

import (
	"encoding/json"
	"net/http"

	"chat-backend/services"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(s *services.AuthService) *AuthHandler { return &AuthHandler{svc: s} }

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, "Method not allowed", "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON", "Bad request format", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		respondWithError(w, "Missing fields", "Username and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.svc.Register(req.Username, req.Password)
	if err != nil {
		respondWithError(w, "Registration failed", err.Error(), http.StatusBadRequest)
		return
	}

	token, err := h.svc.CreateToken(user.ID, user.Username)
	if err != nil {
		respondWithError(w, "Token creation failed", "Could not create authentication token", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"token": token,
		"user":  user,
	}

	respondWithSuccess(w, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, "Method not allowed", "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON", "Bad request format", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		respondWithError(w, "Missing fields", "Username and password are required", http.StatusBadRequest)
		return
	}

	token, user, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		respondWithError(w, "Authentication failed", err.Error(), http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"token": token,
		"user":  user,
	}

	respondWithSuccess(w, response)
}

func respondWithError(w http.ResponseWriter, error, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   error,
		Message: message,
	})
}

func respondWithSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
	})
}
