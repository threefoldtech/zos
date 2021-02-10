package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/versioned"
)

const (
	pathByID   = "by-id"
	pathByType = "by-type"
	pathByUser = "by-user"
)

var (
	// workloadSchemaV1 reservation schema version 1
	workloadSchemaV1 = versioned.MustParse("1.0.0")
	// ReservationSchemaLastVersion link to latest version
	workloadSchemaLastVersion = workloadSchemaV1

	typeIDfn = map[gridtypes.WorkloadType]func(*gridtypes.Workload) (string, error){
		gridtypes.NetworkType: networkTypeID,
	}
)

func networkTypeID(w *gridtypes.Workload) (string, error) {
	var name struct {
		Name string `json:"name"`
	}

	err := json.Unmarshal(w.Data, &name)
	if err != nil {
		return "", errors.Wrap(err, "failed to load network name")
	}
	if len(name.Name) == 0 {
		return "", fmt.Errorf("empty network name")
	}

	return string(gridtypes.NetworkID(w.User.String(), name.Name)), nil
}

// Fs is a in reservation cache using the filesystem as backend
type Fs struct {
	m    sync.RWMutex
	root string
}

var _ (provision.Storage) = (*Fs)(nil)

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

func (s *Fs) pathByType(wl *gridtypes.Workload) (string, error) {
	// by types is different than by id in 2 aspects
	// 1- it prefix the path with `by-type/<type>`
	// 2- the second and not so obvious difference is that
	// it doesn't use the workload.ID instead it uses the id
	// calculated based on the type. It falls back to workload.ID
	// if no custom id method for that type.
	id := wl.ID.String()
	fn, ok := typeIDfn[wl.Type]
	if ok {
		var err error
		id, err = fn(wl)
		if err != nil {
			return "", err
		}
	}

	return filepath.Join(pathByType, wl.Type.String(), id), nil
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
	s.m.Lock()
	defer s.m.Unlock()

	byType, err := s.pathByType(&wl)
	if err != nil {
		return err
	}

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
		s.rooted(byType),
		s.rooted(s.pathByUser(&wl), s.pathByID(&wl)),
		s.rooted(s.pathByUser(&wl), byType),
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
	s.m.Lock()
	defer s.m.Unlock()

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
func (s *Fs) get(path string) (gridtypes.Workload, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	var wl gridtypes.Workload
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

// Get gets a workload by id
func (s *Fs) Get(id gridtypes.ID) (gridtypes.Workload, error) {
	path := s.rooted(filepath.Join(pathByID, id.String()))
	return s.get(path)
}

func (s *Fs) byType(base string, t gridtypes.WorkloadType) ([]gridtypes.ID, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	dir := filepath.Join(base, pathByType, t.String())
	entries, err := ioutil.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var results []gridtypes.ID
	for _, entry := range entries {
		if entry.Mode()&os.ModeSymlink == 0 {
			continue
		}
		target, err := os.Readlink(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		results = append(results, gridtypes.ID(filepath.Base(target)))
	}

	return results, nil
}

// ByType return list of reservation ids by type
func (s *Fs) ByType(t gridtypes.WorkloadType) ([]gridtypes.ID, error) {
	return s.byType(s.root, t)
}

// ByUser return list of reservation for a certain user by type
func (s *Fs) ByUser(user gridtypes.ID, t gridtypes.WorkloadType) ([]gridtypes.ID, error) {
	base := filepath.Join(s.root, pathByUser, user.String())
	return s.byType(base, t)
}

// GetNetwork returns network object given network id
func (s *Fs) GetNetwork(id gridtypes.NetID) (gridtypes.Workload, error) {
	path := filepath.Join(s.root, pathByType, gridtypes.NetworkType.String(), id.String())
	return s.get(path)
}

// Close makes sure the backend of the store is closed properly
func (s *Fs) Close() error {
	return nil
}