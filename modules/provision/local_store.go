package provision

import (
	"fmt"
	"sync"
	"time"
)

type memStore struct {
	sync.RWMutex
	m map[string]*Reservation
}

func NewMemStore() *memStore {
	return &memStore{
		m: make(map[string]*Reservation),
	}
}

// Add a reservation ID to the store
func (s *memStore) Add(r *Reservation) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.m[r.ID]; ok {
		return fmt.Errorf("reservation %s already in the store", r.ID)
	}

	s.m[r.ID] = r
	return nil
}

// Remove a reservation ID from the store
func (s *memStore) Remove(id string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.m[id]; ok {
		delete(s.m, id)
	}

	return nil
}

// GetExpired returns all id the the reservations that are expired
// at the time of the function call
func (s *memStore) GetExpired() ([]*Reservation, error) {
	s.RLock()
	defer s.RUnlock()

	output := make([]*Reservation, 0, len(s.m)/2)
	for _, r := range s.m {
		if !isExpired(r) {
			continue
		}
		output = append(output, r)
	}
	return output, nil
}

func isExpired(r *Reservation) bool {
	expire := r.Created.Add(r.Duration)
	return time.Now().After(expire)
}

// Exits checks if a reservation id is already present in the store
func (s *memStore) Get(id string) (*Reservation, error) {
	s.Lock()
	defer s.Unlock()

	r, ok := s.m[id]
	if !ok {
		return nil, fmt.Errorf("reservation not found")
	}
	return r, nil
}

// Close makes sure the backend of the store is closed properly
func (s *memStore) Close() error {
	return nil
}
