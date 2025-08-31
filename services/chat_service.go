package services

import (
	"errors"

	"chat-backend/models"
	"chat-backend/repository"
)

type ChatService struct {
	chats repository.ChatRepository
	users repository.UserRepository
}

func NewChatService(cr repository.ChatRepository, ur repository.UserRepository) *ChatService {
	return &ChatService{chats: cr, users: ur}
}

func (s *ChatService) CreateRoom(name string) (*models.ChatRoom, error) {
	if name == "" {
		return nil, errors.New("room name cannot be empty")
	}
	if len(name) < 2 {
		return nil, errors.New("room name too short (minimum 2 characters)")
	}
	if len(name) > 50 {
		return nil, errors.New("room name too long (maximum 50 characters)")
	}
	
	return s.chats.Create(name)
}

func (s *ChatService) ListRooms() ([]models.ChatRoom, error) {
	return s.chats.List()
}

func (s *ChatService) GetRoomByID(roomID int) (*models.ChatRoom, error) {
	return s.chats.FindByID(roomID)
}
