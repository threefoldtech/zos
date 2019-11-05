package filesystem

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	require := require.New(t)
	var exec TestExecuter
	mgr := TestDeviceManager{
		devices: DeviceCache{
			Device{Path: "/tmp/fakea", Label: "1234"},
			Device{Path: "/tmp/fakeb", Label: "sp_1234"},
			Device{Path: "/tmp/fakec", Children: []Device{
				Device{Path: "/tmp/fakec1", Label: "sp_1234"},
			}},
			Device{Path: "/tmp/faked", Children: []Device{
				Device{Path: "/tmp/faked1", Label: "1234"},
			}},
			Device{Path: "/tmp/fakee"},
		},
	}

	ctx := context.Background()
	// we expect calls to wipe for /tmp/fakeb and /tmp/fakec
	exec.On("run", ctx, "parted", "-s", "/tmp/fakeb", "mktable", "msdos").Return([]byte{}, nil)
	exec.On("run", ctx, "parted", "-s", "/tmp/fakec", "mktable", "msdos").Return([]byte{}, nil)

	exec.On("run", ctx, "sync").Return([]byte{}, nil)
	exec.On("run", ctx, "partprobe").Return([]byte{}, nil)
	_, err := migrate(ctx, &mgr, &exec)
	require.NoError(err)

	exec.AssertCalled(t, "run", ctx, "parted", "-s", "/tmp/fakeb", "mktable", "msdos")
	exec.AssertCalled(t, "run", ctx, "parted", "-s", "/tmp/fakec", "mktable", "msdos")
	exec.AssertNotCalled(t, "run", ctx, "parted", "-s", "/tmp/fakea", "mktable", "msdos")
	exec.AssertNotCalled(t, "run", ctx, "parted", "-s", "/tmp/faked", "mktable", "msdos")
	exec.AssertNotCalled(t, "run", ctx, "parted", "-s", "/tmp/fakee", "mktable", "msdos")

	for _, fake := range []string{"/tmp/fakeb", "/tmp/fakec"} {
		stat, err := os.Stat(fake)
		require.NoError(err)
		require.Equal(int64(512), stat.Size())
		os.Remove(fake)
	}
}
