package filesystem

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// NewBcachefsPool creates a btrfs pool associated with device.
// if device does not have a filesystem one is created
func NewBcachefsPool(device DeviceInfo) (Pool, error) {
	return newBcachefsPool(device, executerFunc(run))
}

func newBcachefsPool(device DeviceInfo, exe executer) (Pool, error) {
	pool := &bcachefsPool{
		device: device,
		utils:  newBcachefsCmd(exe),
		name:   device.Label,
	}

	return pool, pool.prepare()
}

var (
	errNotImplemented = errors.New("not implemented")
)

type bcachefsPool struct {
	device DeviceInfo
	utils  bcachefsUtils
	name   string
}

func (p *bcachefsPool) prepare() error {
	// check if already have filesystem
	if p.device.Used() {
		return nil
	}
	ctx := context.TODO()

	// otherwise format
	if err := p.format(ctx); err != nil {
		return err
	}
	// make sure kernel knows about this
	return Partprobe(ctx)
}

func (p *bcachefsPool) format(ctx context.Context) error {
	name := uuid.New().String()
	p.name = name

	args := []string{
		"-L", name,
		p.device.Path,
	}

	if _, err := p.utils.run(ctx, "mkfs.bcachefs", args...); err != nil {
		return errors.Wrapf(err, "failed to format device '%s'", p.device.Path)
	}

	return nil
}

// Volume ID
func (b *bcachefsPool) ID() int {
	return 0
}

// Path of the volume
func (b *bcachefsPool) Path() string {
	return filepath.Join("/mnt", b.name)
}

// Usage returns the pool usage
func (b *bcachefsPool) Usage() (Usage, error) {
	return Usage{}, errNotImplemented
}

// Limit on a pool is not supported yet
func (b *bcachefsPool) Limit(size uint64) error {
	return errNotImplemented
}

// Name of the volume
func (b *bcachefsPool) Name() string {
	return b.name
}

// FsType of the volume
func (b *bcachefsPool) FsType() string {
	return "bcachefs"
}

// Mounted returns whether the pool is mounted or not. If it is mounted,
// the mountpoint is returned
// It doesn't check the default mount location of the pool
// but instead check if any of the pool devices is mounted
// under any location
func (p *bcachefsPool) Mounted() (string, error) {
	ctx := context.TODO()
	mnt, err := p.device.Mountpoint(ctx)
	if err != nil {
		return "", err
	}

	if len(mnt) != 0 {
		return mnt, nil
	}

	return "", ErrDeviceNotMounted
}

// Mount the pool, the mountpoint is returned
func (p *bcachefsPool) Mount() (string, error) {
	mnt, err := p.Mounted()
	if err == nil {
		return mnt, nil
	} else if !errors.Is(err, ErrDeviceNotMounted) {
		return "", errors.Wrap(err, "failed to check device mount status")
	}

	// device is not mounted
	mnt = p.Path()
	if err := os.MkdirAll(mnt, 0755); err != nil {
		return "", err
	}

	if err := syscall.Mount(p.device.Path, mnt, "bcachefs", 0, ""); err != nil {
		return "", err
	}

	// TODO: check
	//if err := p.utils.QGroupEnable(ctx, mnt); err != nil {
	//	return "", fmt.Errorf("failed to enable qgroup: %w", err)
	//}

	return mnt, p.maintenance()
}

func (p *bcachefsPool) maintenance() error {
	return errNotImplemented
}

// UnMount the pool
func (p *bcachefsPool) UnMount() error {
	mnt, err := p.Mounted()
	if err != nil {
		if errors.Is(err, ErrDeviceNotMounted) {
			return nil
		}
		return err
	}

	return syscall.Unmount(mnt, syscall.MNT_DETACH)
}

// Volumes are all subvolumes of this volume
// TODO: bcachefs doesn't have the feature
func (p *bcachefsPool) Volumes() ([]Volume, error) {
	return nil, errNotImplemented
}

// AddVolume adds a new subvolume with the given name
func (p *bcachefsPool) AddVolume(name string) (Volume, error) {
	mnt, err := p.Mounted()
	if err != nil {
		return nil, err
	}

	root := filepath.Join(mnt, name)
	return p.addVolume(root)
}

func (p *bcachefsPool) addVolume(root string) (Volume, error) {
	ctx := context.Background()
	if err := p.utils.SubvolumeAdd(ctx, root); err != nil {
		return nil, err
	}

	//volume, err := p.utils.SubvolumeInfo(ctx, root)
	//if err != nil {
	//	return nil, err
	//}
	return &bcachefsVolume{
		id:   0,
		path: root,
	}, nil
}

// RemoveVolume removes a subvolume with the given name
func (b *bcachefsPool) RemoveVolume(name string) error {
	mnt, err := b.Mounted()
	if err != nil {
		return err
	}

	root := filepath.Join(mnt, name)
	return b.removeVolume(root)
}

func (p *bcachefsPool) removeVolume(root string) error {
	ctx := context.Background()

	//info, err := p.utils.SubvolumeInfo(ctx, root)
	//if err != nil {
	//	return err
	//}

	if err := p.utils.SubvolumeRemove(ctx, root); err != nil {
		return err
	}

	/*qgroupID := fmt.Sprintf("0/%d", info.ID)
	if err := p.utils.QGroupDestroy(ctx, qgroupID, p.Path()); err != nil {
		// we log here and not return an error because
		// - qgroup deletion can fail because it is still used by the system
		//   even if the volume is gone
		// - failure to delete a qgroup is not a fatal error
		log.Warn().Err(err).Str("group-id", qgroupID).Msg("failed to delete qgroup")
		return nil
	}*/

	return nil
}

// Shutdown spins down the device where the pool is mounted
func (b *bcachefsPool) Shutdown() error {
	cmd := exec.Command("hdparm", "-y", b.device.Path)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to shutdown device '%s'", b.device.Path)
	}

	return nil
}

// Device return device associated with pool
func (b *bcachefsPool) Device() DeviceInfo {
	return b.device
}

// SetType sets a device type on the pool. this will make
// sure that the detected device type is reported
// correctly by calling the Type() method.
// TODO : merge the code with btrfs
func (b *bcachefsPool) SetType(typ zos.DeviceType) error {
	path, err := b.Mounted()
	if err != nil {
		return err
	}
	diskTypePath := filepath.Join(path, ".seektime")
	if err := os.WriteFile(diskTypePath, []byte(typ), 0644); err != nil {
		return errors.Wrapf(err, "failed to store device type for '%s' in '%s'", b.Name(), diskTypePath)
	}

	return nil
}

// Type returns the device type set by a previous call
// to SetType.
// TODO : merge the code with btrfs
func (b *bcachefsPool) Type() (zos.DeviceType, bool, error) {
	path, err := b.Mounted()
	if err != nil {
		return "", false, err
	}
	diskTypePath := filepath.Join(path, ".seektime")
	diskType, err := os.ReadFile(diskTypePath)
	if os.IsNotExist(err) {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	if len(diskType) == 0 {
		return "", false, nil
	}

	return zos.DeviceType(diskType), true, nil
}

type bcachefsVolume struct {
	id   int
	path string
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
	err = errNotImplemented
	return
}

// Limit size of volume, setting size to 0 means unlimited
func (v *bcachefsVolume) Limit(size uint64) error {
	return errNotImplemented
}
