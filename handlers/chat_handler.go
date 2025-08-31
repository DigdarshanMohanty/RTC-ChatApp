package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"chat-backend/services"
	"chat-backend/ws"
)

type ChatHandler struct {
	hub     *ws.Hub
	chatSvc *services.ChatService
	authSvc *services.AuthService
	msgSvc  *services.MessageService
}

func NewChatHandler(h *ws.Hub, c *services.ChatService, a *services.AuthService, m *services.MessageService) *ChatHandler {
	return &ChatHandler{hub: h, chatSvc: c, authSvc: a, msgSvc: m}
}

// Middleware for authentication
func (h *ChatHandler) WithAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			// Fallback for WebSocket clients
			proto := r.Header.Get("Sec-WebSocket-Protocol")
			if proto != "" {
				token = proto
			}
		}
		if token == "" {
			respondWithError(w, "Unauthorized", "Missing Authorization header (token only)", http.StatusUnauthorized)
			return
		}
		uid, uname, err := h.authSvc.ParseToken(token)
		if err != nil {
			respondWithError(w, "Unauthorized", "Invalid token", http.StatusUnauthorized)
			return
		}
		r.Header.Set("X-User-ID", strconv.Itoa(uid))
		r.Header.Set("X-Username", uname)
		next(w, r)
	}
}

// List chat rooms
func (h *ChatHandler) Rooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, "Method not allowed", "Use GET method", http.StatusMethodNotAllowed)
		return
	}

	rooms, err := h.chatSvc.ListRooms()
	if err != nil {
		respondWithError(w, "Internal error", "Failed to list rooms", http.StatusInternalServerError)
		return
	}

	respondWithSuccess(w, rooms)
}

// Create a chat room
func (h *ChatHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, "Method not allowed", "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON", "Bad request format", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondWithError(w, "Missing name", "Room name is required", http.StatusBadRequest)
		return
	}

	room, err := h.chatSvc.CreateRoom(req.Name)
	if err != nil {
		respondWithError(w, "Room creation failed", err.Error(), http.StatusBadRequest)
		return
	}

	respondWithSuccess(w, room)
}

// WebSocket handler
func (h *ChatHandler) WS(w http.ResponseWriter, r *http.Request) {
	log.Printf("WebSocket connection attempt from %s", r.RemoteAddr)
	log.Printf("Request headers: %+v", r.Header)
	log.Printf("Request URL: %s", r.URL.String())

	roomIDStr := r.URL.Query().Get("roomId")
	if roomIDStr == "" {
		log.Printf("WebSocket connection rejected: missing roomId parameter")
		respondWithError(w, "Missing parameter", "roomId query parameter is required", http.StatusBadRequest)
		return
	}

	// Extract token from URL query parameters for WebSocket connections
	token := r.URL.Query().Get("token")
	if token == "" {
		log.Printf("WebSocket connection rejected: missing token parameter")
		respondWithError(w, "Missing parameter", "token query parameter is required", http.StatusBadRequest)
		return
	}

	// Validate the token
	uid, uname, err := h.authSvc.ParseToken(token)
	if err != nil {
		log.Printf("WebSocket connection rejected: invalid token - %v", err)
		respondWithError(w, "Unauthorized", "Invalid token", http.StatusUnauthorized)
		return
	}

	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		log.Printf("WebSocket connection rejected: invalid roomId '%s' - %v", roomIDStr, err)
		respondWithError(w, "Invalid parameter", "roomId must be a valid number", http.StatusBadRequest)
		return
	}

	log.Printf("WebSocket connection validated for user %s (ID: %d) in room %d", uname, uid, roomID)
	h.hub.ServeWS(w, r, roomID, uid, uname, h.msgSvc)
}
