package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/provision"
)

type reservation struct {
	Reservation *provision.Reservation `json:"reservation"`
	Result      *provision.Result      `json:"result"`
	Deleted     bool                   `json:"deleted"`
	NodeID      string                 `json:"node_id"`
}

type provisionStore struct {
	Reservations []*reservation `json:"reservations"`
	m            sync.RWMutex
}

func LoadProvisionStore() (nodeStore, error) {
	store := nodeStore{
		Nodes: []*directory.TfgridNode2{},
	}
	f, err := os.OpenFile("reservations.json", os.O_RDONLY, 0660)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		return store, err
	}
	return store, nil
}

func (s *provisionStore) Save() error {
	s.m.RLock()
	defer s.m.RUnlock()

	f, err := os.OpenFile("reservations.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(s); err != nil {
		return err
	}
	return nil
}

func (s *provisionStore) List() []*reservation {
	s.m.RLock()
	defer s.m.RUnlock()
	out := make([]*reservation, len(s.Reservations))

	copy(out, s.Reservations)
	return out
}

func (s *provisionStore) Get(ID string) (*reservation, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	for _, r := range s.Reservations {
		if r.Reservation.ID == ID {
			return r, nil
		}
	}

	return nil, fmt.Errorf("reservation %s not found", ID)
}

func (s *provisionStore) Add(r reservation) error {
	s.m.Lock()
	defer s.m.Unlock()

	s.Reservations = append(s.Reservations, &r)
	return nil
}
