package cache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
	"github.com/threefoldtech/zos/pkg/versioned"
)

var (
	// reservationSchemaV1 reservation schema version 1
	reservationSchemaV1 = versioned.MustParse("1.0.0")
	// ReservationSchemaLastVersion link to latest version
	reservationSchemaLastVersion = reservationSchemaV1
)

// Filter is filtering function for Purge method
type Filter func(*provision.Reservation) bool

// FilterType delete all types that matches the given list
func FilterType(types ...provision.ReservationType) Filter {
	typeMap := make(map[provision.ReservationType]struct{})
	for _, t := range types {
		typeMap[t] = struct{}{}
	}

	return func(r *provision.Reservation) bool {
		_, ok := typeMap[r.Type]

		return ok
	}
}

// FilterNot inverts the given filter
func FilterNot(f Filter) Filter {
	return func(r *provision.Reservation) bool {
		return !f(r)
	}
}

var (
	// NotPersisted filter outs everything but volumes
	NotPersisted = FilterNot(FilterType(provision.VolumeReservation))
)

// Fs is a in reservation cache using the filesystem as backend
type Fs struct {
	m    sync.RWMutex
	root string
}

// NewFSStore creates a in memory reservation store
func NewFSStore(root string) (*Fs, error) {
	store := &Fs{
		root: root,
	}

	if err := os.MkdirAll(root, 0770); err != nil {
		return nil, err
	}

	if err := store.updateReservationResults(root); err != nil {
		log.Error().Err(err).Msgf("error while updating reservation results")
		return store, nil
	}

	return store, nil
}

// Updates reservation results for reservations in cache that don't have a result set.
func (s *Fs) updateReservationResults(rootPath string) error {
	log.Info().Msg("updating reservation results")
	reservations, err := s.List()
	if err != nil {
		return err
	}

	client, err := app.ExplorerClient()
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		if !reservation.Result.IsNil() {
			continue
		}

		log.Info().Msgf("updating reservation result for %s", reservation.ID)

		result, err := client.Workloads.NodeWorkloadGet(reservation.ID)
		if err != nil {
			log.Error().Err(err).Msgf("error occurred while requesting reservation result for %s", reservation.ID)
			continue
		}

		provisionResult := result.GetResult()
		reservation.Result = provision.Result{
			Type:      reservation.Type,
			Created:   provisionResult.Epoch.Time,
			State:     provision.ResultState(provisionResult.State),
			Data:      provisionResult.DataJson,
			Error:     provisionResult.Message,
			ID:        provisionResult.WorkloadId,
			Signature: provisionResult.Signature,
		}

		err = s.add(reservation, true)
		if err != nil {
			log.Error().Err(err).Msg("error while updating reservation in cache")
			continue
		}
	}

	return nil
}

// Purge deletes all cached reservations that matches filter
func (s *Fs) Purge(f Filter) error {
	s.m.Lock()
	s.m.Unlock()

	// if rootPath is not present on the filesystem, return
	_, err := os.Stat(s.root)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err = filepath.Walk(s.root, func(path string, info os.FileInfo, r error) error {
		if r != nil {
			return r
		}
		// if a file with size 0 is present we can assume its empty and remove it
		if info.Size() == 0 {
			log.Warn().Str("filename", info.Name()).Msg("cached reservation found, but file is empty, removing.")
			return os.Remove(path)
		}

		if info.IsDir() {
			return nil
		}
		id := filepath.Base(path)
		reservation, err := s.get(id)
		if err != nil {
			return err
		}
		if f(reservation) {
			log.Info().Str("reservation", reservation.ID).Msg("removing cached reservation")
			return s.remove(id)
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "error scanning cached reservations")
	}
	return nil
}

// CurrentCounters gets current capacity counters from cache
func (s *Fs) CurrentCounters() (primitives.Counters, error) {
	var counter primitives.Counters
	if err := s.sync(&counter); err != nil {
		return counter, err
	}

	return counter, nil
}

// sync update the statser with all the reservation present in the cache
func (s *Fs) sync(statser provision.Counter) error {
	s.m.RLock()
	defer s.m.RUnlock()

	return s.incrementCounters(statser)
}

// Add a reservation to the store
func (s *Fs) Add(r *provision.Reservation, override bool) error {
	return s.add(r, override)
}

// Add a reservation to the store
func (s *Fs) add(r *provision.Reservation, override bool) error {
	s.m.Lock()
	defer s.m.Unlock()

	// ensure direcory exists
	if err := os.MkdirAll(s.root, 0770); err != nil {
		return err
	}

	flags := os.O_CREATE | os.O_WRONLY
	if !override {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(filepath.Join(s.root, r.ID), flags, 0660)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("reservation %s already in the store", r.ID)
		}
		return err
	}
	defer f.Close()
	writer, err := versioned.NewWriter(f, reservationSchemaLastVersion)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(writer).Encode(r); err != nil {
		return err
	}

	return nil
}

// Remove a reservation from the store
func (s *Fs) Remove(id string) error {
	s.m.Lock()
	defer s.m.Unlock()

	return s.remove(id)
}

func (s *Fs) remove(id string) error {
	path := filepath.Join(s.root, id)
	err := os.Remove(path)
	if os.IsNotExist(errors.Cause(err)) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

// GetExpired returns all id the the reservations that are expired
// at the time of the function call
func (s *Fs) GetExpired() ([]*provision.Reservation, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	infos, err := ioutil.ReadDir(s.root)
	if err != nil {
		return nil, err
	}

	rs := make([]*provision.Reservation, 0, len(infos))
	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		// if the file is empty, remove it and return.
		if info.Size() == 0 {
			if info.Size() == 0 {
				log.Warn().Str("filename", info.Name()).Msg("cached reservation found, but file is empty, removing.")
				return nil, os.Remove(path.Join(s.root, info.Name()))
			}
		}

		r, err := s.get(info.Name())
		if err != nil {
			return nil, err
		}
		if r.Expired() {
			// r.Tag = Tag{"source": "FSStore"}
			rs = append(rs, r)
		}

	}

	return rs, nil
}

// Get retrieves a specific reservation using its ID
// if returns a non nil error if the reservation is not present in the store
func (s *Fs) Get(id string) (*provision.Reservation, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	return s.get(id)
}

// Exists checks if the reservation ID is in the store
func (s *Fs) Exists(id string) (bool, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	path := filepath.Join(s.root, id)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// NetworkExists exists checks if a network exists in cache already
func (s *Fs) NetworkExists(id string) (bool, error) {
	reservations, err := s.List()
	if err != nil {
		return false, err
	}

	for _, r := range reservations {
		if r.Type == primitives.NetworkReservation {
			nr := pkg.NetResource{}
			if err := json.Unmarshal(r.Data, &nr); err != nil {
				return false, fmt.Errorf("failed to unmarshal network from reservation: %w", err)
			}

			// Check if the combination of network id and user is the same
			if string(provision.NetworkID(r.User, nr.Name)) == id {
				return true, nil
			}
		}
	}

	return false, nil
}

// List all reservations
func (s *Fs) List() ([]*provision.Reservation, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	infos, err := ioutil.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	reservations := make([]*provision.Reservation, 0, len(infos))

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		r, err := s.get(info.Name())
		if err != nil {
			return nil, fmt.Errorf("failed get reservation: %w", err)
		}

		reservations = append(reservations, r)
	}
	return reservations, nil
}

// incrementCounters will increment counters for all workloads
// for network workloads it will only increment those that have a unique name
func (s *Fs) incrementCounters(statser provision.Counter) error {
	uniqueNetworkReservations := make(map[pkg.NetID]*provision.Reservation)

	reservations, err := s.List()
	if err != nil {
		return err
	}

	for _, r := range reservations {
		if r.Expired() || r.Result.State != provision.StateOk {
			continue
		}
		if r.Type == primitives.NetworkResourceReservation || r.Type == primitives.NetworkReservation {
			nr := pkg.NetResource{}
			if err := json.Unmarshal(r.Data, &nr); err != nil {
				return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
			}

			netID := provision.NetworkID(r.User, nr.Name)
			// if the network name + user exsists in the list, we skip it.
			// else we add it to the list
			if _, ok := uniqueNetworkReservations[netID]; ok {
				continue
			}

			uniqueNetworkReservations[netID] = r
			continue
		} else {
			if err := statser.Increment(r); err != nil {
				return fmt.Errorf("fail to update stats:%w", err)
			}
		}
	}

	for _, r := range uniqueNetworkReservations {
		if err := statser.Increment(r); err != nil {
			return fmt.Errorf("fail to update stats:%w", err)
		}
	}
	return nil
}

func (s *Fs) get(id string) (*provision.Reservation, error) {
	path := filepath.Join(s.root, id)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "reservation %s not found", id)
	} else if err != nil {
		return nil, err
	}

	defer f.Close()
	reader, err := versioned.NewReader(f)
	if err != nil && versioned.IsNotVersioned(err) {
		if _, err := f.Seek(0, 0); err != nil { // make sure to read from start
			return nil, err
		}
		reader = versioned.NewVersionedReader(versioned.MustParse("0.0.0"), f)
	}
	if err != nil {
		return nil, err
	}

	validV1 := versioned.MustParseRange(fmt.Sprintf("<=%s", reservationSchemaV1))
	var reservation provision.Reservation

	if validV1(reader.Version()) {
		if err := json.NewDecoder(reader).Decode(&reservation); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown reservation object version (%s)", reader.Version())
	}
	// reservation.Tag = Tag{"source": "FSStore"}
	return &reservation, nil
}

// Close makes sure the backend of the store is closed properly
func (s *Fs) Close() error {
	return nil
}
