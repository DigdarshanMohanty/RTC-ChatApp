package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type ChatRoom struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	IsPrivate  bool      `json:"is_private"`
	InviteCode string    `json:"invite_code,omitempty"`
	CreatedBy  int       `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// GenerateInviteCode generates a secure random invite code
func GenerateInviteCode() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// RoomMembership represents user access to private rooms
type RoomMembership struct {
	ID       int       `json:"id"`
	RoomID   int       `json:"room_id"`
	UserID   int       `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}
