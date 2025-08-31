package repository

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"chat-backend/models"
)

type MembershipRepository interface {
	AddMember(roomID, userID int) error
	RemoveMember(roomID, userID int) error
	IsUserMember(roomID, userID int) (bool, error)
	GetRoomMembers(roomID int) ([]int, error)
	GetUserRooms(userID int) ([]int, error)
	GetMembershipsByRoom(roomID int) ([]models.RoomMembership, error)
}

type InMemoryMembershipRepo struct {
	mu   sync.RWMutex
	seq  int
	data map[int]*models.RoomMembership // by membership id
	byRU map[string]int                 // "roomID:userID" -> membership id
}

func NewInMemoryMembershipRepo() *InMemoryMembershipRepo {
	return &InMemoryMembershipRepo{
		data: make(map[int]*models.RoomMembership),
		byRU: make(map[string]int),
	}
}

func (r *InMemoryMembershipRepo) AddMember(roomID, userID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := formatRoomUserKey(roomID, userID)
	if _, exists := r.byRU[key]; exists {
		return nil // Already a member
	}

	r.seq++
	membership := &models.RoomMembership{
		ID:       r.seq,
		RoomID:   roomID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}

	r.data[membership.ID] = membership
	r.byRU[key] = membership.ID

	return nil
}

func (r *InMemoryMembershipRepo) RemoveMember(roomID, userID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := formatRoomUserKey(roomID, userID)
	membershipID, exists := r.byRU[key]
	if !exists {
		return errors.New("membership not found")
	}

	delete(r.data, membershipID)
	delete(r.byRU, key)

	return nil
}

func (r *InMemoryMembershipRepo) IsUserMember(roomID, userID int) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := formatRoomUserKey(roomID, userID)
	_, exists := r.byRU[key]
	return exists, nil
}

func (r *InMemoryMembershipRepo) GetRoomMembers(roomID int) ([]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var members []int
	for _, membership := range r.data {
		if membership.RoomID == roomID {
			members = append(members, membership.UserID)
		}
	}

	return members, nil
}

func (r *InMemoryMembershipRepo) GetUserRooms(userID int) ([]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var rooms []int
	for _, membership := range r.data {
		if membership.UserID == userID {
			rooms = append(rooms, membership.RoomID)
		}
	}

	return rooms, nil
}

func (r *InMemoryMembershipRepo) GetMembershipsByRoom(roomID int) ([]models.RoomMembership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var memberships []models.RoomMembership
	for _, membership := range r.data {
		if membership.RoomID == roomID {
			memberships = append(memberships, *membership)
		}
	}

	return memberships, nil
}

func formatRoomUserKey(roomID, userID int) string {
	return fmt.Sprintf("%d:%d", roomID, userID)
}
