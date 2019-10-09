/*
NOTE:
  This test file tries to create loop devices to work against on some
  sparse files, to avoid causing any actual changes to permanent storage
  on the test machine. This comes with a price that this test file
  need to run as root to be able to run the `losetup` commands.

  this can be easily done as
  `sudo GOPATH=$GOPATH go test -v ./...`

   (
	  we set the gopath to the current user gopath to make it use the same go cache and src from the
	  current user
   )
*/
package filesystem

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"testing"

	"github.com/threefoldtech/zos/pkg"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type TestDeviceManager struct {
	devices DeviceCache
}

func (m *TestDeviceManager) Device(ctx context.Context, path string) (*Device, error) {
	for idx := range m.devices {
		loop := &m.devices[idx]
		if loop.Path == path {
			return loop, nil
		}
	}

	return nil, fmt.Errorf("device not found")
}

func (m *TestDeviceManager) ByLabel(ctx context.Context, label string) (DeviceCache, error) {
	var filterred DeviceCache
	for _, device := range m.devices {
		if device.Label == label {
			filterred = append(filterred, device)
		}
	}
	return filterred, nil
}

func (m *TestDeviceManager) Devices(ctx context.Context) (DeviceCache, error) {
	return m.devices, nil
}

func basePoolTest(t *testing.T, exec *TestExecuter, pool Pool) {
	const tmp = `Label: '%s'  uuid: 081717ad-77d5-488a-afd0-ab9108784f70
Total devices 1 FS bytes used 206665822208
devid    1 size 462713520128 used 211548110848 path /dev/sdb2
`

	l := fmt.Sprintf(tmp, pool.Name())
	exec.On("run", mock.Anything, "btrfs", "filesystem", "show", "--raw", pool.Name()).
		Return([]byte(l), nil)

	t.Run("test mounted", func(t *testing.T) {
		// mounted will check if the pool (fs) is listed
		// with -m
		exec.On("run", mock.Anything, "btrfs", "filesystem", "show", "--raw", "-m", pool.Name()).
			Return([]byte{}, nil)
		// return an empty list will mean false.
		_, mounted := pool.Mounted()
		assert.False(t, mounted)
	})

	t.Run("test path", func(t *testing.T) {
		assert.Equal(t, path.Join("/mnt", pool.Name()), pool.Path())
	})

	t.Run("test mount", func(t *testing.T) {
		t.Skip("we can't skip this fake pool")
		// mount device
		target, err := pool.Mount()
		assert.NoError(t, err)

		assert.Equal(t, target, pool.Path())
	})

	exec.ExpectedCalls = nil
	// now make sure to return it as mounted
	exec.On("run", mock.Anything, "btrfs", "filesystem", "show", "--raw", "-m", pool.Name()).
		Return([]byte(l), nil)

	exec.On("run", mock.Anything, "btrfs", "subvolume", "list", "-o", "/").
		Return([]byte{}, nil)

	t.Run("test no subvolumes", func(t *testing.T) {
		// no volumes
		volumes, err := pool.Volumes()
		require.NoError(t, err)

		assert.Empty(t, volumes)
	})

	var volume Volume
	var err error
	t.Run("test create volume", func(t *testing.T) {
		mnt := filepath.Join(pool.Path(), "subvol1")

		exec.On("run", mock.Anything, "btrfs", "subvolume", "create", mnt).
			Return(nil, nil)

		volume, err = pool.AddVolume("subvol1")
		require.NoError(t, err)

		assert.Equal(t, path.Join("/mnt", pool.Name(), "subvol1"), volume.Path())
	})

	t.Run("test list volumes", func(t *testing.T) {
		volumes, err := pool.Volumes()
		require.NoError(t, err)

		assert.Len(t, volumes, 1)
	})

	t.Run("test usage", func(t *testing.T) {
		usage, err := pool.Usage()
		require.NoError(t, err)
		assert.Equal(t, uint64(1024*1024*1024), usage.Size)
	})

	t.Run("test subvolume list no subvolumes", func(t *testing.T) {
		volumes, err := volume.Volumes()
		require.NoError(t, err)

		assert.Empty(t, volumes)
	})

	t.Run("test limit subvolume", func(t *testing.T) {
		usage, err := volume.Usage()
		require.NoError(t, err)

		// Note: an empty subvolume has an overhead of 16384 bytes
		assert.Equal(t, Usage{Used: 16384}, usage)

		err = volume.Limit(50 * 1024 * 1024)
		require.NoError(t, err)

		usage, err = volume.Usage()
		require.NoError(t, err)

		// Note: an empty subvolume has an overhead of 16384 bytes
		assert.Equal(t, Usage{Used: 16384, Size: 50 * 1024 * 1024}, usage)
	})

	t.Run("test remove subvolume", func(t *testing.T) {
		err = pool.RemoveVolume("subvol1")
		require.NoError(t, err)
		// no volumes after delete
		volumes, err := pool.Volumes()
		require.NoError(t, err)

		assert.Empty(t, volumes)
	})
}

func TestBtrfsCreateSingle(t *testing.T) {
	require := require.New(t)
	mgr := &TestDeviceManager{
		devices: DeviceCache{
			Device{Path: "/tmp/dev1", DiskType: pkg.SSDDevice},
		},
	}

	var exec TestExecuter

	exec.On("run", mock.Anything, "mkfs.btrfs", "-L", "test-single", "-d", "single", "-m", "single", "/tmp/dev1").
		Return([]byte{}, nil)

	fs := newBtrfs(mgr, &exec)
	_, err := fs.Create(context.Background(), "test-single", mgr.devices, pkg.Single)
	require.NoError(err)

	require.Equal("test-single", mgr.devices[0].Label)
	require.Equal(BtrfsFSType, mgr.devices[0].Filesystem)

	//basePoolTest(t, &exec, pool)
}

func TestBtrfsCreateRaid1(t *testing.T) {
	require := require.New(t)
	mgr := &TestDeviceManager{
		devices: DeviceCache{
			Device{Path: "/tmp/dev1", DiskType: pkg.SSDDevice},
			Device{Path: "/tmp/dev2", DiskType: pkg.SSDDevice},
		},
	}

	var exec TestExecuter

	exec.On("run", mock.Anything, "mkfs.btrfs", "-L", "test-raid1",
		"-d", "raid1", "-m", "raid1",
		"/tmp/dev1", "/tmp/dev2").Return([]byte{}, nil)

	fs := newBtrfs(mgr, &exec)
	_, err := fs.Create(context.Background(), "test-raid1", mgr.devices, pkg.Raid1)
	require.NoError(err)

	require.Equal("test-raid1", mgr.devices[0].Label)
	require.Equal(BtrfsFSType, mgr.devices[0].Filesystem)

	require.Equal("test-raid1", mgr.devices[1].Label)
	require.Equal(BtrfsFSType, mgr.devices[1].Filesystem)
}
