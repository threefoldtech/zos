package primitives

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
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/versioned"
)

var (
	// reservationSchemaV1 reservation schema version 1
	reservationSchemaV1 = versioned.MustParse("1.0.0")
	// ReservationSchemaLastVersion link to latest version
	reservationSchemaLastVersion = reservationSchemaV1
)

// FSStore is a in reservation store
// using the filesystem as backend
type FSStore struct {
	sync.RWMutex
	root string
}

// NewFSStore creates a in memory reservation store
func NewFSStore(root string) (*FSStore, error) {
	store := &FSStore{
		root: root,
	}
	if app.IsFirstBoot("provisiond") {
		log.Info().Msg("first boot, empty reservation cache")
		if err := store.removeAllButPersistent(root); err != nil {
			return nil, err
		}

		if err := app.MarkBooted("provisiond"); err != nil {
			return nil, errors.Wrap(err, "fail to mark provisiond as booted")
		}
	}

	if err := os.MkdirAll(root, 0770); err != nil {
		return nil, err
	}

	log.Info().Msg("restart detected, keep reservation cache intact")

	return store, nil
}

//TODO: i think both sync and removeAllButPersistent can be merged into
// one method because now it scans the same directory twice.
func (s *FSStore) removeAllButPersistent(rootPath string) error {
	// if rootPath is not present on the filesystem, return
	_, err := os.Stat(rootPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, r error) error {
		if r != nil {
			return r
		}
		// if a file with size 0 is present we can assume its empty and remove it
		if info.Size() == 0 {
			log.Warn().Str("filename", info.Name()).Msg("cached reservation %d found, but file is empty, removing.")
			return os.Remove(path)
		}

		if info.IsDir() {
			return nil
		}
		reservationType, err := s.getType(filepath.Base(path))
		if err != nil {
			return err
		}
		if reservationType != VolumeReservation && reservationType != ZDBReservation {
			log.Info().Msgf("Removing %s from cache", path)
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		log.Error().Msgf("error walking the path %q: %v\n", rootPath, err)
		return err
	}
	return nil
}

func (s *FSStore) Sync(statser provision.Statser) error {
	s.RLock()
	defer s.RUnlock()

	infos, err := ioutil.ReadDir(s.root)
	if err != nil {
		return err
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		r, err := s.get(info.Name())
		if err != nil {
			return err
		}

		statser.Increment(r)
	}

	return nil
}

// Add a reservation to the store
func (s *FSStore) Add(r *provision.Reservation) error {
	s.Lock()
	defer s.Unlock()

	// ensure direcory exists
	if err := os.MkdirAll(s.root, 0770); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(s.root, r.ID), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
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

	// s.counterFor(r.Type).Increment(1)
	// if err := s.processResourceUnits(r, true); err != nil {
	// 	return errors.Wrapf(err, "could not compute the amount of resource used by reservation %s", r.ID)
	// }

	return nil
}

// Remove a reservation from the store
func (s *FSStore) Remove(id string) error {
	s.Lock()
	defer s.Unlock()

	// r, err := s.get(id)
	// if os.IsNotExist(errors.Cause(err)) {
	// 	return nil
	// }

	path := filepath.Join(s.root, id)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		// shouldn't happen because we just did a get
		return nil
	} else if err != nil {
		return err
	}

	// s.counterFor(r.Type).Decrement(1)
	// if err := s.processResourceUnits(r, false); err != nil {
	// 	return errors.Wrapf(err, "could not compute the amount of resource used by reservation %s", r.ID)
	// }

	return nil
}

// GetExpired returns all id the the reservations that are expired
// at the time of the function call
func (s *FSStore) GetExpired() ([]*provision.Reservation, error) {
	s.RLock()
	defer s.RUnlock()

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
				log.Warn().Str("filename", info.Name()).Msg("cached reservation %d found, but file is empty, removing.")
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
func (s *FSStore) Get(id string) (*provision.Reservation, error) {
	s.RLock()
	defer s.RUnlock()

	return s.get(id)
}

// getType retrieves a specific reservation's type using its ID
// if returns a non nil error if the reservation is not present in the store
func (s *FSStore) getType(id string) (provision.ReservationType, error) {
	res := struct {
		Type provision.ReservationType `json:"type"`
	}{}
	path := filepath.Join(s.root, id)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return "", errors.Wrapf(err, "reservation %s not found", id)
	} else if err != nil {
		return "", err
	}

	defer f.Close()
	reader, err := versioned.NewReader(f)
	if versioned.IsNotVersioned(err) {
		if _, err := f.Seek(0, 0); err != nil { // make sure to read from start
			return "", err
		}
		reader = versioned.NewVersionedReader(versioned.MustParse("0.0.0"), f)
	}

	validV1 := versioned.MustParseRange(fmt.Sprintf("<=%s", reservationSchemaV1))

	if validV1(reader.Version()) {
		if err := json.NewDecoder(reader).Decode(&res); err != nil {
			return "nil", err
		}
	} else {
		return "", fmt.Errorf("unknown reservation object version (%s)", reader.Version())
	}
	return res.Type, nil
}

// Exists checks if the reservation ID is in the store
func (s *FSStore) Exists(id string) (bool, error) {
	s.RLock()
	defer s.RUnlock()

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

func (s *FSStore) get(id string) (*provision.Reservation, error) {
	path := filepath.Join(s.root, id)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "reservation %s not found", id)
	} else if err != nil {
		return nil, err
	}

	defer f.Close()
	reader, err := versioned.NewReader(f)
	if versioned.IsNotVersioned(err) {
		if _, err := f.Seek(0, 0); err != nil { // make sure to read from start
			return nil, err
		}
		reader = versioned.NewVersionedReader(versioned.MustParse("0.0.0"), f)
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
func (s *FSStore) Close() error {
	return nil
}
