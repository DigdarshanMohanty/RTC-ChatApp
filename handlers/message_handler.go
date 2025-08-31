package handlers

import (
	"net/http"
	"strconv"

	"chat-backend/services"
)

type MessageHandler struct {
	svc     *services.MessageService
	authSvc *services.AuthService
}

func NewMessageHandler(s *services.MessageService, a *services.AuthService) *MessageHandler {
	return &MessageHandler{svc: s, authSvc: a}
}

func (h *MessageHandler) WithAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			respondWithError(w, "Unauthorized", "Missing Authorization header (token only)", http.StatusUnauthorized)
			return
		}
		uid, _, err := h.authSvc.ParseToken(token)
		if err != nil {
			respondWithError(w, "Unauthorized", "Invalid token", http.StatusUnauthorized)
			return
		}
		// stash user id in context if you want; for now pass along header
		r.Header.Set("X-User-ID", strconv.Itoa(uid))
		next(w, r)
	}
}

func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, "Method not allowed", "Use GET method", http.StatusMethodNotAllowed)
		return
	}
	
	roomIDStr := r.URL.Query().Get("roomId")
	if roomIDStr == "" {
		respondWithError(w, "Missing parameter", "roomId query parameter is required", http.StatusBadRequest)
		return
	}
	
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		respondWithError(w, "Invalid parameter", "roomId must be a valid number", http.StatusBadRequest)
		return
	}
	
	// Get limit from query params, default to 50
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	msgs, err := h.svc.List(roomID, limit)
	if err != nil {
		respondWithError(w, "Failed to fetch messages", err.Error(), http.StatusBadRequest)
		return
	}
	
	respondWithSuccess(w, msgs)
}
