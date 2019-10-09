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
	"testing"

	"github.com/threefoldtech/zos/pkg"

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
