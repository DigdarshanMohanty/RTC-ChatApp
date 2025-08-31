package repository

import (
	"errors"
	"sync"
	"time"

	"chat-backend/models"
)

type ChatRepository interface {
	Create(name string, isPrivate bool, createdBy int) (*models.ChatRoom, error)
	List() ([]models.ChatRoom, error)
	ListAccessibleRooms(userID int, membershipRepo MembershipRepository) ([]models.ChatRoom, error)
	FindByID(id int) (*models.ChatRoom, error)
	FindByInviteCode(inviteCode string) (*models.ChatRoom, error)
	GetUserCount(roomID int) (int, error)
	Delete(id int) error
	CanUserAccess(roomID, userID int, membershipRepo MembershipRepository) (bool, error)
}

type InMemoryChatRepo struct {
	mu   sync.RWMutex
	seq  int
	data map[int]*models.ChatRoom
}

func NewInMemoryChatRepo() *InMemoryChatRepo {
	return &InMemoryChatRepo{
		data: make(map[int]*models.ChatRoom),
	}
}

func (r *InMemoryChatRepo) Create(name string, isPrivate bool, createdBy int) (*models.ChatRoom, error) {
	if name == "" {
		return nil, errors.New("room name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate names
	for _, room := range r.data {
		if room.Name == name {
			return nil, errors.New("room name already exists")
		}
	}

	r.seq++
	room := &models.ChatRoom{
		ID:        r.seq,
		Name:      name,
		IsPrivate: isPrivate,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	// Generate invite code for private rooms
	if isPrivate {
		room.InviteCode = models.GenerateInviteCode()
	}

	r.data[room.ID] = room
	return room, nil
}

func (r *InMemoryChatRepo) List() ([]models.ChatRoom, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rooms := make([]models.ChatRoom, 0, len(r.data))
	for _, v := range r.data {
		rooms = append(rooms, *v)
	}
	return rooms, nil
}

func (r *InMemoryChatRepo) ListAccessibleRooms(userID int, membershipRepo MembershipRepository) ([]models.ChatRoom, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var accessibleRooms []models.ChatRoom
	for _, room := range r.data {
		if !room.IsPrivate {
			// Public rooms are accessible to everyone
			accessibleRooms = append(accessibleRooms, *room)
		} else {
			// Check if user is member of private room or is the creator
			if room.CreatedBy == userID {
				accessibleRooms = append(accessibleRooms, *room)
			} else {
				isMember, err := membershipRepo.IsUserMember(room.ID, userID)
				if err == nil && isMember {
					accessibleRooms = append(accessibleRooms, *room)
				}
			}
		}
	}
	return accessibleRooms, nil
}

func (r *InMemoryChatRepo) FindByID(id int) (*models.ChatRoom, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	room, ok := r.data[id]
	if !ok {
		return nil, errors.New("room not found")
	}
	return room, nil
}

func (r *InMemoryChatRepo) GetUserCount(roomID int) (int, error) {
	// This would need to be implemented with the websocket hub
	// For now, return 0 as placeholder
	return 0, nil
}

func (r *InMemoryChatRepo) FindByInviteCode(inviteCode string) (*models.ChatRoom, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, room := range r.data {
		if room.InviteCode == inviteCode {
			return room, nil
		}
	}
	return nil, errors.New("room not found")
}

func (r *InMemoryChatRepo) CanUserAccess(roomID, userID int, membershipRepo MembershipRepository) (bool, error) {
	r.mu.RLock()
	room, ok := r.data[roomID]
	r.mu.RUnlock()

	if !ok {
		return false, errors.New("room not found")
	}

	// Public rooms are accessible to everyone
	if !room.IsPrivate {
		return true, nil
	}

	// Creator has access
	if room.CreatedBy == userID {
		return true, nil
	}

	// Check membership for private rooms
	return membershipRepo.IsUserMember(roomID, userID)
}

func (r *InMemoryChatRepo) Delete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[id]; !ok {
		return errors.New("room not found")
	}

	delete(r.data, id)
	return nil
}
