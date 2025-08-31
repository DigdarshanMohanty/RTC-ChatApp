package services

import (
	"errors"
	"strconv"
	"time"

	"chat-backend/config"
	"chat-backend/models"
	"chat-backend/repository"
)

// MessageBroadcaster interface to avoid import cycles
type MessageBroadcaster interface {
	BroadcastMessage(msg models.Message, username string)
}

type MessageService struct {
	msgs   repository.MessageRepository
	chats  repository.ChatRepository
	users  repository.UserRepository
	hub    MessageBroadcaster
	config *config.Config
}

func NewMessageService(mr repository.MessageRepository, cr repository.ChatRepository, ur repository.UserRepository, hub MessageBroadcaster, cfg *config.Config) *MessageService {
	return &MessageService{msgs: mr, chats: cr, users: ur, hub: hub, config: cfg}
}

func (s *MessageService) Send(roomID, senderID int, content string) (*models.Message, error) {
	if content == "" {
		return nil, errors.New("empty content")
	}
	if len(content) > s.config.MaxMessageLength {
		return nil, errors.New("message too long (max " + strconv.Itoa(s.config.MaxMessageLength) + " characters)")
	}

	if _, err := s.chats.FindByID(roomID); err != nil {
		return nil, errors.New("room not found")
	}

	user, err := s.users.FindByID(senderID)
	if err != nil {
		return nil, errors.New("sender not found")
	}

	msg := &models.Message{
		RoomID:    roomID,
		SenderID:  senderID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	saved, err := s.msgs.Save(msg)
	if err != nil {
		return nil, err
	}

	// broadcast over hub with username
	s.hub.BroadcastMessage(*saved, user.Username)
	return saved, nil
}

func (s *MessageService) List(roomID int, limit int) ([]models.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	if _, err := s.chats.FindByID(roomID); err != nil {
		return nil, errors.New("room not found")
	}

	msgs, err := s.msgs.ListByRoom(roomID, limit)
	if err != nil {
		return nil, err
	}

	// Populate username for each message
	for i := range msgs {
		user, err := s.users.FindByID(msgs[i].SenderID)
		if err != nil {
			msgs[i].Username = "Unknown User"
		} else {
			msgs[i].Username = user.Username
		}
	}

	return msgs, nil
}
