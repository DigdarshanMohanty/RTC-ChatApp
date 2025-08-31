package repository

import (
	"errors"
	"sync"
	"time"

	"chat-backend/models"
)

type UserRepository interface {
	Create(username, hashedPwd string) (*models.User, error)
	FindByUsername(username string) (*models.User, error)
	FindByID(id int) (*models.User, error)
}

type InMemoryUserRepo struct {
	mu    sync.RWMutex
	seq   int
	byID  map[int]*models.User
	byU   map[string]*models.User
}

func NewInMemoryUserRepo() *InMemoryUserRepo {
	return &InMemoryUserRepo{
		byID: make(map[int]*models.User),
		byU:  make(map[string]*models.User),
	}
}

func (r *InMemoryUserRepo) Create(username, hashedPwd string) (*models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byU[username]; ok {
		return nil, errors.New("username already exists")
	}
	r.seq++
	u := &models.User{
		ID:        r.seq,
		Username:  username,
		Password:  hashedPwd,
		CreatedAt: time.Now(),
	}
	r.byID[u.ID] = u
	r.byU[u.Username] = u
	return u, nil
}

func (r *InMemoryUserRepo) FindByUsername(username string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byU[username]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (r *InMemoryUserRepo) FindByID(id int) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}
