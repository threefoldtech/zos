package filesystem

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/stretchr/testify/assert"
)

type TestDevices map[string]string

func (d TestDevices) Loops() []string {
	var loops []string
	for _, loop := range d {
		loops = append(loops, loop)
	}

	return loops
}

func (d TestDevices) Destroy() {
	for file, loop := range d {
		if loop != "" {
			run(context.Background(), "losetup", "-d", loop)
		}

		if err := os.Remove(file); err != nil {
			fmt.Printf("failed to delete file '%s': %s\n", file, err)
		}
	}
}

func setupDevices(count int) (devices TestDevices, err error) {
	devices = make(TestDevices)

	defer func() {
		if err != nil {
			devices.Destroy()
		}
	}()

	for i := 0; i < count; i++ {
		var dev *os.File
		dev, err = ioutil.TempFile("", "loop-test-")
		if err != nil {
			return
		}

		devices[dev.Name()] = ""

		if err = dev.Truncate(1024 * 1024 * 1024); err != nil { // 1G
			return
		}

		if err = dev.Close(); err != nil {
			return
		}
		var output []byte
		output, err = run(context.Background(), "losetup", "-f", "--show", dev.Name())
		if err != nil {
			return
		}

		devices[dev.Name()] = strings.TrimSpace(string(output))
	}

	return
}

type TestDeviceManager struct {
	devices TestDevices
}

func (m TestDeviceManager) Devices(ctx context.Context) ([]Device, error) {
	var devices []Device
	for loop := range m.devices {
		devices = append(devices, Device{
			Path: loop,
			Name: path.Base(loop),
			Type: "loop",
		})
	}

	return devices, nil
}

func TestMain(m *testing.M) {
	defer func() {
		//make sure to try to detach all remaining loop devices from testing
		run(context.Background(), "losetup", "-D")
	}()

	os.Exit(m.Run())
}

func TestBtrfsCreateSingle(t *testing.T) {
	devices, err := setupDevices(1)
	if err != nil {
		t.Fatal("failed to initialize devices", err)
	}
	defer devices.Destroy()

	fs := NewBtrfs(TestDeviceManager{devices})

	var pool Pool
	t.Run("create pool", func(t *testing.T) {
		pool, err = fs.Create(context.Background(), "test-single", devices.Loops(), modules.Single)

		if ok := assert.NoError(t, err); !ok {
			t.Fail()
		}
	})

	t.Run("test mounted", func(t *testing.T) {
		_, mounted := pool.Mounted()
		if ok := assert.False(t, mounted); !ok {
			t.Error()
		}
	})

	t.Run("test path", func(t *testing.T) {
		if ok := assert.Equal(t, "/mnt/test-single", pool.Path()); !ok {
			t.Error()
		}
	})

	t.Run("test mount", func(t *testing.T) {
		// mount device
		target, err := pool.Mount()
		if ok := assert.NoError(t, err); !ok {
			t.Fail()
		}

		if ok := assert.Equal(t, target, pool.Path()); !ok {
			t.Error()
		}
	})

	defer pool.UnMount()

	t.Run("test no subvolumes", func(t *testing.T) {
		// no volumes
		volumes, err := pool.Volumes()
		if ok := assert.NoError(t, err); !ok {
			t.Fail()
		}

		if ok := assert.Empty(t, volumes); !ok {
			t.Error()
		}
	})

	var volume Volume
	t.Run("test create volume", func(t *testing.T) {
		volume, err = pool.AddVolume("subvol1", 0)
		if ok := assert.NoError(t, err); !ok {
			t.Fail()
		}

		if ok := assert.Equal(t, "/mnt/test-single/subvol1", volume.Path()); !ok {
			t.Error()
		}
	})

	t.Run("test list volumes", func(t *testing.T) {
		volumes, err := pool.Volumes()
		if ok := assert.NoError(t, err); !ok {
			t.Fail()
		}

		if ok := assert.Len(t, volumes, 1); !ok {
			t.Error()
		}
	})

	t.Run("test subvolume list no subvolumes", func(t *testing.T) {
		volumes, err := volume.Volumes()
		if ok := assert.NoError(t, err); !ok {
			t.Fail()
		}

		if ok := assert.Empty(t, volumes); !ok {
			t.Error()
		}
	})

	t.Run("test remove subvolume", func(t *testing.T) {
		err = pool.RemoveVolume("subvol1")
		if ok := assert.NoError(t, err); !ok {
			t.Fail()
		}
		// no volumes after delete
		volumes, err := pool.Volumes()
		if ok := assert.NoError(t, err); !ok {
			t.Fail()
		}

		if ok := assert.Empty(t, volumes); !ok {
			t.Error()
		}
	})
}
