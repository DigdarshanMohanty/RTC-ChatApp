package repository

import (
	"errors"
	"sync"
	"time"

	"chat-backend/models"
)

type ChatRepository interface {
	Create(name string) (*models.ChatRoom, error)
	List() ([]models.ChatRoom, error)
	FindByID(id int) (*models.ChatRoom, error)
	GetUserCount(roomID int) (int, error)
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

func (r *InMemoryChatRepo) Create(name string) (*models.ChatRoom, error) {
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
		CreatedAt: time.Now(),
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
