package repository

import (
	"errors"
	"sort"
	"sync"
	"time"

	"chat-backend/models"
)

type MessageRepository interface {
	Save(msg *models.Message) (*models.Message, error)
	ListByRoom(roomID int, limit int) ([]models.Message, error)
}

type InMemoryMessageRepo struct {
	mu   sync.RWMutex
	seq  int
	data map[int]*models.Message // by id
	byR  map[int][]int           // room -> message IDs
}

func NewInMemoryMessageRepo() *InMemoryMessageRepo {
	return &InMemoryMessageRepo{
		data: make(map[int]*models.Message),
		byR:  make(map[int][]int),
	}
}

func (r *InMemoryMessageRepo) Save(msg *models.Message) (*models.Message, error) {
	if msg == nil {
		return nil, errors.New("nil message")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	msg.ID = r.seq
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	r.data[msg.ID] = msg
	r.byR[msg.RoomID] = append(r.byR[msg.RoomID], msg.ID)
	return msg, nil
}

func (r *InMemoryMessageRepo) ListByRoom(roomID int, limit int) ([]models.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byR[roomID]
	if len(ids) == 0 {
		return []models.Message{}, nil
	}
	// collect and sort by createdAt (implicitly ID order here)
	var msgs []models.Message
	for _, id := range ids {
		m := r.data[id]
		msgs = append(msgs, *m)
	}
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].CreatedAt.Before(msgs[j].CreatedAt)
	})
	if limit > 0 && len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}
