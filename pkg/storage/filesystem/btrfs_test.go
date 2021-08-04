package filesystem

// import (
// 	"context"
// 	"fmt"
// 	"testing"

// 	"github.com/threefoldtech/zos/pkg"
// 	"github.com/threefoldtech/zos/pkg/gridtypes/zos"

// 	"github.com/stretchr/testify/mock"
// 	"github.com/stretchr/testify/require"
// )

// type TestDeviceManager struct {
// 	devices Devices
// }

// func (m *TestDeviceManager) Reset() DeviceManager {
// 	return m
// }

// func (m *TestDeviceManager) Device(ctx context.Context, path string) (Device, error) {
// 	for _, loop := range m.devices {
// 		if loop.Path() == path {
// 			return loop, nil
// 		}
// 	}

// 	return nil, fmt.Errorf("device not found")
// }

// func (m *TestDeviceManager) ByLabel(ctx context.Context, label string) ([]Device, error) {
// 	var filterred []Device
// 	for _, device := range m.devices {
// 		info, err := device.Info()
// 		if err != nil {
// 			return nil, err
// 		}

// 		if info.Label == label {
// 			filterred = append(filterred, device)
// 		}
// 	}

// 	return filterred, nil
// }

// func (m *TestDeviceManager) Devices(ctx context.Context) (Devices, error) {
// 	return m.devices, nil
// }

// func (m *TestDeviceManager) Raw(ctx context.Context) (Devices, error) {
// 	return m.devices, nil
// }

// func TestBtrfsCreateSingle(t *testing.T) {
// 	require := require.New(t)
// 	mgr := &TestDeviceManager{
// 		devices: Devices{
// 			DeviceImpl{Path: "/tmp/dev1", DiskType: zos.SSDDevice},
// 		},
// 	}

// 	var exec TestExecuter

// 	exec.On("run", mock.Anything, "mkfs.btrfs", "-L", "test-single", "-d", "single", "-m", "single", "/tmp/dev1").
// 		Return([]byte{}, nil)

// 	fs := newBtrfs(mgr, &exec)
// 	_, err := fs.Create(context.Background(), "test-single", pkg.Single, &mgr.devices[0])
// 	require.NoError(err)

// 	require.Equal("test-single", mgr.devices[0].Label)
// 	require.Equal(BtrfsFSType, mgr.devices[0].Filesystem)

// 	//basePoolTest(t, &exec, pool)
// }
