package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type bcachefsUtils struct {
	executer
	volManager bcachefsVolumeManager
}

func newBcachefsCmd(exec executer) bcachefsUtils {
	return bcachefsUtils{
		executer:   exec,
		volManager: newBcachefsVolumeManager(),
	}
}

// SubvolumeAdd adds a new subvolume at path
func (u *bcachefsUtils) SubvolumeAdd(ctx context.Context, root string) (bcachefsVolume, error) {
	_, err := u.run(ctx, "bcachefs", "subvolume", "create", root)
	if err != nil {
		return bcachefsVolume{}, err
	}
	return u.volManager.Add(root)
}

// SubvolumeRemove removes a subvolume
func (u *bcachefsUtils) SubvolumeRemove(ctx context.Context, root string) error {
	_, err := u.run(ctx, "bcachefs", "subvolume", "delete", root)
	if err != nil {
		return err
	}
	return u.volManager.Delete(root)
}

func (u *bcachefsUtils) SubvolumeList(ctx context.Context, root string) ([]bcachefsVolume, error) {
	return u.volManager.list(root)
}

func (u *bcachefsUtils) SubvolumeInfo(ctx context.Context, root string) (bcachefsVolume, error) {
	return u.volManager.Get(root)
}

// bcachefs volume menager is a hack for the minimal support of sublvolume in bcachefs.
// it does several things:
// - keep track of the subvolumes created by the driver
// - listing subvolumes
//
// current implementation is considered a hack and should be replaced with a proper implementation
// before going to production
type bcachefsVolumeManager struct {
	//root string
	//vols map[string]bcachefsVolume
}

type bcachefsSubvolumes map[string]bcachefsVolume

func (vols bcachefsSubvolumes) add(vol bcachefsVolume) {
	vols[vol.Name()] = vol
}

func (vols bcachefsSubvolumes) get(root string) (bcachefsVolume, bool) {
	v, ok := vols[root]
	return v, ok
}

func (vols bcachefsSubvolumes) delete(root string) {
	delete(vols, root)
}

func newBcachefsVolumeManager() bcachefsVolumeManager {
	return bcachefsVolumeManager{}
}

func (m bcachefsVolumeManager) metaFile(root string) string {
	const (
		metaFileName = ".volumes"
	)
	return filepath.Join(filepath.Dir(root), metaFileName)
}
func (m bcachefsVolumeManager) Add(root string) (bcachefsVolume, error) {
	log.Info().Str("root", root).Msg("Add volume")
	vols, err := m.load(root)
	if err != nil {
		return bcachefsVolume{}, fmt.Errorf("Add failed: %v", err)
	}
	vol := newBcachefsVol(0, root, 0, m)
	vols.add(vol)
	return vol, m.save(root, vols)
}

func (m bcachefsVolumeManager) Delete(root string) error {
	log.Info().Str("root", root).Msg("Delete volume")
	vols, err := m.load(root)
	if err != nil {
		return fmt.Errorf("Delete failed: %v", err)
	}
	vols.delete(root)
	return m.save(root, vols)
}

func (m bcachefsVolumeManager) Get(root string) (vol bcachefsVolume, err error) {
	log.Info().Str("root", root).Msg("Get volume")
	vols, err := m.load(root)
	if err != nil {
		err = fmt.Errorf("failed to get volume %v: %v", root, err)
		return
	}
	vol, ok := vols.get(root)
	if !ok {
		err = fmt.Errorf("volume %s not found", root)
		return
	}
	vol.mgr = m
	return
}

func (m bcachefsVolumeManager) Set(vol bcachefsVolume) error {
	root := vol.path
	log.Info().Str("root", root).Msg("Set volume")
	vols, err := m.load(root)
	if err != nil {
		return fmt.Errorf("Set failed: %v", err)
	}
	vols[vol.Name()] = vol
	return m.save(root, vols)
}

func (m bcachefsVolumeManager) list(root string) ([]bcachefsVolume, error) {
	log.Info().Str("root", root).Msg("List volume")
	vols, err := m.load(root + "/x") // TODO fix this ugly append hack
	if err != nil {
		return nil, fmt.Errorf("list failed: %v", err)
	}
	res := make([]bcachefsVolume, 0, len(vols))
	for name, v := range vols {
		v.path = name
		res = append(res, v)
	}
	return res, nil
}

func (m bcachefsVolumeManager) load(root string) (bcachefsSubvolumes, error) {
	f, err := os.OpenFile(m.metaFile(root), os.O_RDONLY, 0644)
	if err != nil {
		log.Error().Err(err).Str("root", root).Msg("failed to open .volumes file")
		if !m.isMetaExists(root) {
			log.Info().Msg("meta not exists")
			return bcachefsSubvolumes{}, m.initialize(root)
		}
		log.Info().Msg("meta exists")
		return nil, err
	}
	defer f.Close()
	vols := bcachefsSubvolumes{}
	err = json.NewDecoder(f).Decode(&vols)
	return vols, err
}

func (m *bcachefsVolumeManager) isMetaExists(root string) bool {
	// Use os.Stat to get file information
	_, err := os.Stat(m.metaFile(root))
	return !os.IsNotExist(err)
}

func (m *bcachefsVolumeManager) initialize(root string) error {
	return m.save(root, bcachefsSubvolumes{})
}

func (m bcachefsVolumeManager) save(root string, vols bcachefsSubvolumes) error {
	f, err := os.OpenFile(m.metaFile(root), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(vols)
}

type bcachefsVolume struct {
	id   int
	path string
	Size uint64
	mgr  bcachefsVolumeManager
}

func newBcachefsVol(id int, path string, size uint64, mgr bcachefsVolumeManager) bcachefsVolume {
	return bcachefsVolume{
		id:   id,
		path: path,
		Size: size,
		mgr:  mgr,
	}
}

func (v *bcachefsVolume) ToStorageVolume(mnt string) Volume {
	return &bcachefsVolume{
		id:   v.id,
		path: filepath.Join(mnt, v.Name()),
		Size: v.Size,
		mgr:  v.mgr,
	}
}

func (v *bcachefsVolume) ID() int {
	return v.id
}

func (v *bcachefsVolume) Path() string {
	return v.path
}

// Name of the filesystem
func (v *bcachefsVolume) Name() string {
	return filepath.Base(v.Path())
}

// FsType of the filesystem
func (v *bcachefsVolume) FsType() string {
	return "bcachefs"
}

// Usage return the volume usage
func (v *bcachefsVolume) Usage() (usage Usage, err error) {
	used := v.Size
	if used == 0 {
		// in case no limit is set on the subvolume, we assume
		// it's size is the size of the files on that volumes
		// or a special case when the volume is a zdb volume
		used, err = volumeUsage(v.Path())
		if err != nil {
			return usage, errors.Wrap(err, "failed to get subvolume usage")
		}
	}

	return Usage{Used: used, Size: v.Size, Excl: 0}, nil
}

// Limit size of volume, setting size to 0 means unlimited
func (v *bcachefsVolume) Limit(size uint64) error {
	v.Size = size
	return v.mgr.Set(*v)
}
