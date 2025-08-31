package services

import (
	"errors"

	"chat-backend/models"
	"chat-backend/repository"
)

type ChatService struct {
	chats       repository.ChatRepository
	users       repository.UserRepository
	messages    repository.MessageRepository
	memberships repository.MembershipRepository
}

func NewChatService(cr repository.ChatRepository, ur repository.UserRepository, mr repository.MessageRepository, memRepo repository.MembershipRepository) *ChatService {
	return &ChatService{chats: cr, users: ur, messages: mr, memberships: memRepo}
}

func (s *ChatService) CreateRoom(name string, isPrivate bool, createdBy int) (*models.ChatRoom, error) {
	if name == "" {
		return nil, errors.New("room name cannot be empty")
	}
	if len(name) < 2 {
		return nil, errors.New("room name too short (minimum 2 characters)")
	}
	if len(name) > 50 {
		return nil, errors.New("room name too long (maximum 50 characters)")
	}

	room, err := s.chats.Create(name, isPrivate, createdBy)
	if err != nil {
		return nil, err
	}

	// Add creator as member of private room
	if isPrivate {
		err = s.memberships.AddMember(room.ID, createdBy)
		if err != nil {
			return nil, errors.New("failed to add creator to room")
		}
	}

	return room, nil
}

func (s *ChatService) ListRooms() ([]models.ChatRoom, error) {
	return s.chats.List()
}

func (s *ChatService) ListAccessibleRooms(userID int) ([]models.ChatRoom, error) {
	return s.chats.ListAccessibleRooms(userID, s.memberships)
}

func (s *ChatService) GetRoomByID(roomID int) (*models.ChatRoom, error) {
	return s.chats.FindByID(roomID)
}

func (s *ChatService) JoinRoomByInvite(inviteCode string, userID int) (*models.ChatRoom, error) {
	room, err := s.chats.FindByInviteCode(inviteCode)
	if err != nil {
		return nil, errors.New("invalid invite code")
	}

	if !room.IsPrivate {
		return nil, errors.New("invite codes are only for private rooms")
	}

	// Add user as member
	err = s.memberships.AddMember(room.ID, userID)
	if err != nil {
		return nil, errors.New("failed to join room")
	}

	return room, nil
}

func (s *ChatService) CanUserAccessRoom(roomID, userID int) (bool, error) {
	return s.chats.CanUserAccess(roomID, userID, s.memberships)
}

func (s *ChatService) DeleteRoom(roomID int, userID int) error {
	// Prevent deletion of the default room (ID 1)
	if roomID == 1 {
		return errors.New("cannot delete the default room")
	}

	// Check if room exists and user can delete it
	room, err := s.chats.FindByID(roomID)
	if err != nil {
		return err // Room not found
	}

	// Only room creator can delete the room
	if room.CreatedBy != userID {
		return errors.New("only room creator can delete the room")
	}

	// Delete all memberships for this room
	members, err := s.memberships.GetRoomMembers(roomID)
	if err == nil {
		for _, memberID := range members {
			s.memberships.RemoveMember(roomID, memberID)
		}
	}

	// Delete all messages in the room
	if err := s.messages.DeleteByRoom(roomID); err != nil {
		return errors.New("failed to delete messages from room")
	}

	// Delete the room
	if err := s.chats.Delete(roomID); err != nil {
		return err
	}

	return nil
}
