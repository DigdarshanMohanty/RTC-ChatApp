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

	// Get user ID from header
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondWithError(w, "Invalid user", "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get accessible rooms for this user
	rooms, err := h.chatSvc.ListAccessibleRooms(userID)
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
		Name      string `json:"name"`
		IsPrivate bool   `json:"is_private"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON", "Bad request format", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondWithError(w, "Missing name", "Room name is required", http.StatusBadRequest)
		return
	}

	// Get user ID from header
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondWithError(w, "Invalid user", "Invalid user ID", http.StatusBadRequest)
		return
	}

	room, err := h.chatSvc.CreateRoom(req.Name, req.IsPrivate, userID)
	if err != nil {
		respondWithError(w, "Room creation failed", err.Error(), http.StatusBadRequest)
		return
	}

	respondWithSuccess(w, room)
}

// Delete a chat room
func (h *ChatHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		respondWithError(w, "Method not allowed", "Use DELETE method", http.StatusMethodNotAllowed)
		return
	}

	roomIDStr := r.URL.Query().Get("id")
	if roomIDStr == "" {
		respondWithError(w, "Missing parameter", "Room id is required", http.StatusBadRequest)
		return
	}

	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		respondWithError(w, "Invalid parameter", "Room id must be a valid number", http.StatusBadRequest)
		return
	}

	// Get user ID from header
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondWithError(w, "Invalid user", "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := h.chatSvc.DeleteRoom(roomID, userID); err != nil {
		respondWithError(w, "Room deletion failed", err.Error(), http.StatusBadRequest)
		return
	}

	// Disconnect all clients from the deleted room
	h.hub.DisconnectRoom(roomID)

	respondWithSuccess(w, map[string]string{"message": "Room deleted successfully"})
}

// Join room via invite code
func (h *ChatHandler) JoinViaInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, "Method not allowed", "Use POST method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		InviteCode string `json:"invite_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON", "Bad request format", http.StatusBadRequest)
		return
	}

	if req.InviteCode == "" {
		respondWithError(w, "Missing invite code", "Invite code is required", http.StatusBadRequest)
		return
	}

	// Get user ID from header
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondWithError(w, "Invalid user", "Invalid user ID", http.StatusBadRequest)
		return
	}

	room, err := h.chatSvc.JoinRoomByInvite(req.InviteCode, userID)
	if err != nil {
		respondWithError(w, "Join failed", err.Error(), http.StatusBadRequest)
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

	// Check if user can access this room
	canAccess, err := h.chatSvc.CanUserAccessRoom(roomID, uid)
	if err != nil {
		log.Printf("WebSocket connection rejected: error checking room access - %v", err)
		respondWithError(w, "Access error", "Failed to verify room access", http.StatusInternalServerError)
		return
	}

	if !canAccess {
		log.Printf("WebSocket connection rejected: user %s (ID: %d) cannot access room %d", uname, uid, roomID)
		respondWithError(w, "Access denied", "You don't have access to this room", http.StatusForbidden)
		return
	}

	log.Printf("WebSocket connection validated for user %s (ID: %d) in room %d", uname, uid, roomID)
	h.hub.ServeWS(w, r, roomID, uid, uname, h.msgSvc)
}
