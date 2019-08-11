package provision

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// LocalStore is the interface use to
// keep a list of reservation and their expiration time
// so provision module can know when to decomission workloads
type LocalStore interface {
	// Add a reservation ID to the store
	Add(r *Reservation) error
	// Remove a reservation ID from to the store
	Remove(id string) error
	// GetExpired returns all id the the reservations that are expired
	// at the time of the function call
	GetExpired() ([]*Reservation, error)
	// Exits checks if a reservation id is already present in the store
	Exists(id string) (bool, error)
	// Close makes sure the backend of the store is closed properly
	Close() error
}

type memStore struct {
	sync.RWMutex
	m map[string]*Reservation
}

func NewMemStore() LocalStore {
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
		log.Debug().Msgf("check reservation %s for expiration", r.ID)
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
func (s *memStore) Exists(id string) (bool, error) {
	s.Lock()
	defer s.Unlock()

	_, ok := s.m[id]
	return ok == true, nil
}

// Close makes sure the backend of the store is closed properly
func (s *memStore) Close() error {
	return nil
}
