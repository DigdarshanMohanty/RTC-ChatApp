package models

import "time"

type Message struct {
	ID         int    `json:"id"`
	SenderID   int    `json:"sender_id"`
	Username   string `json:"username"`
	ReceiverID int    `json:"receiver_id,omitempty"`
	// optional for 1-to-1 chat
	RoomID int `json:"room_id,omitempty"`
	// optional for group chat
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
