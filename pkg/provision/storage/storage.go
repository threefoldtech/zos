package storage

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
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/versioned"
)

var (
	// workloadSchemaV1 reservation schema version 1
	workloadSchemaV1 = versioned.MustParse("1.0.0")
	// ReservationSchemaLastVersion link to latest version
	workloadSchemaLastVersion = workloadSchemaV1
)

// FilterType delete all types that matches the given list
func FilterType(types ...provision.ReservationType) provision.Filter {
	typeMap := make(map[provision.ReservationType]struct{})
	for _, t := range types {
		typeMap[t] = struct{}{}
	}

	return func(r *provision.Reservation) bool {
		_, ok := typeMap[r.Type]

		return ok
	}
}

const (
	pathByID   = "by-id"
	pathByType = "by-type"
	pathByUser = "by-user"
)

// FilterNot inverts the given filter
func FilterNot(f provision.Filter) provision.Filter {
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

	for _, p := range []string{pathByID, pathByType, pathByUser} {
		if err := os.MkdirAll(filepath.Join(root, p), 0770); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (s *Fs) pathByID(wl *gridtypes.Workload) string {
	return filepath.Join(pathByID, wl.ID.String())
}

func (s *Fs) pathByType(wl *gridtypes.Workload) string {
	return filepath.Join(pathByType, wl.Type.String(), wl.ID.String())
}

func (s *Fs) pathByUser(wl *gridtypes.Workload) string {
	return filepath.Join(pathByUser, wl.User.String())
}

func (s *Fs) rooted(p ...string) string {
	return filepath.Join(s.root, filepath.Join(p...))
}

func (s *Fs) symlink(from, to string) error {
	dir := filepath.Dir(from)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create base dir '%s'", dir)
	}

	rel, err := filepath.Rel(filepath.Dir(from), to)
	if err != nil {
		return err
	}

	if err := os.Remove(from); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to cleanup link")
	}

	return os.Symlink(rel, from)
}

// Add workload to database
func (s *Fs) Add(wl gridtypes.Workload) error {
	path := s.rooted(s.pathByID(&wl))
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY|os.O_EXCL,
		0644,
	)

	if os.IsExist(err) {
		return errors.Wrapf(provision.ErrWorkloadExists, "object '%s' exist", wl.ID)
	} else if err != nil {
		return errors.Wrap(err, "failed to open workload file")
	}
	defer file.Close()
	writer, err := versioned.NewWriter(file, workloadSchemaLastVersion)

	if err := json.NewEncoder(writer).Encode(wl); err != nil {
		return errors.Wrap(err, "failed to write workload data")
	}

	for _, link := range []string{
		s.rooted(s.pathByType(&wl)),
		s.rooted(s.pathByUser(&wl), s.pathByID(&wl)),
		s.rooted(s.pathByUser(&wl), s.pathByType(&wl)),
	} {
		if err := s.symlink(link, path); err != nil {
			return err
		}
	}

	return nil
}

// Set updates value of a workload, the reservation must exists
// otherwise an error is returned
func (s *Fs) Set(wl gridtypes.Workload) error {
	path := s.rooted(s.pathByID(&wl))
	file, err := os.OpenFile(
		path,
		os.O_WRONLY,
		0644,
	)
	if os.IsNotExist(err) {
		return errors.Wrapf(provision.ErrWorkloadNotExists, "object '%s' does not exist", wl.ID)
	} else if err != nil {
		return errors.Wrap(err, "failed to open workload file")
	}
	defer file.Close()
	writer, err := versioned.NewWriter(file, workloadSchemaLastVersion)

	if err := json.NewEncoder(writer).Encode(wl); err != nil {
		return errors.Wrap(err, "failed to write workload data")
	}

	return nil
}

// Get gets a workload by id
func (s *Fs) Get(id gridtypes.ID) (gridtypes.Workload, error) {
	var wl gridtypes.Workload
	path := s.rooted(filepath.Join(pathByID, id.String()))
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return wl, errors.Wrapf(provision.ErrWorkloadNotExists, "object '%s' does not exist", wl.ID)
	} else if err != nil {
		return wl, errors.Wrap(err, "failed to open workload file")
	}
	defer file.Close()
	reader, err := versioned.NewReader(file)
	if err != nil {
		return wl, errors.Wrap(err, "failed to load workload")
	}
	version := reader.Version()
	if !version.EQ(workloadSchemaV1) {
		return wl, fmt.Errorf("invalid workload version")
	}

	if err := json.NewDecoder(reader).Decode(&wl); err != nil {
		return wl, errors.Wrap(err, "failed to read workload data")
	}

	return wl, nil
}

// listing
func (s *Fs) ByType(t gridtypes.ReservationType) ([]gridtypes.ID, error) {
	return nil, nil
}
func (s *Fs) ByUser(user gridtypes.ID, t gridtypes.ReservationType) ([]gridtypes.ID, error) {
	return nil, nil

}

func (s *Fs) Network(id gridtypes.NetID) error {
	return nil
}

// Find deletes all cached reservations that matches filter
func (s *Fs) Find(f provision.Filter) ([]*provision.Reservation, error) {
	s.m.Lock()
	s.m.Unlock()

	var results []*provision.Reservation
	// if rootPath is not present on the filesystem, return
	_, err := os.Stat(s.root)
	if os.IsNotExist(err) {
		return results, nil
	} else if err != nil {
		return results, err
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
			results = append(results, reservation)
		}

		return nil
	})

	if err != nil {
		return results, errors.Wrap(err, "error scanning cached reservations")
	}

	return results, nil
}

// Purge deletes all cached reservations that matches filter
func (s *Fs) Purge(f provision.Filter) error {
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
			if err := s.remove(id); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "error scanning cached reservations")
	}
	return nil
}

// // Add a reservation to the store
// func (s *Fs) Add(r *provision.Reservation, override bool) error {
// 	return s.add(r, override)
// }

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
	writer, err := versioned.NewWriter(f, workloadSchemaLastVersion)
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

// // Get retrieves a specific reservation using its ID
// // if returns a non nil error if the reservation is not present in the store
// func (s *Fs) Get(id string) (*provision.Reservation, error) {
// 	s.m.RLock()
// 	defer s.m.RUnlock()

// 	return s.get(id)
// }

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

	validV1 := versioned.MustParseRange(fmt.Sprintf("<=%s", workloadSchemaV1))
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
