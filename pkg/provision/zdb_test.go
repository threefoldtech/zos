package provision

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
)

func TestZDBProvisionExists(t *testing.T) {
	require := require.New(t)

	var client TestClient
	var cache TestZDBCache
	ctx := context.Background()
	ctx = WithZBus(ctx, &client)
	ctx = WithZDBMapping(ctx, &cache)

	const module = "storage"
	version := zbus.ObjectID{Name: "storage", Version: "0.0.1"}

	zdb := ZDB{
		Size:     280682,
		Mode:     pkg.ZDBModeSeq,
		Password: "pa$$w0rd",
		DiskType: pkg.SSDDevice,
		Public:   true,
	}

	reservation := Reservation{
		ID:   "reservation-id",
		User: "user",
		Type: ZDBReservation,
		Data: MustMarshal(t, zdb),
	}

	cache.On("Get", reservation.ID).Return("container-id", true)

	client.On("Request", "container", zbus.ObjectID{Name: "container", Version: "0.0.1"},
		"Inspect", "zdb", pkg.ContainerID("container-id")).
		Return(pkg.Container{
			Name: "container-id",
			Network: pkg.NetworkInfo{
				Namespace: "net-ns",
			},
		}, nil)

	client.On("Request", "network", zbus.ObjectID{Name: "network", Version: "0.0.1"},
		"Addrs",
		"zdb0", "net-ns").Return([]net.IP{net.ParseIP("2001:cdba::3257:9652")}, nil)

	client.On("Request", module, version, "Path", reservation.ID).
		Return("/some/path", nil)

	result, err := zdbProvisionImpl(ctx, &reservation)

	require.NoError(err)
	require.Equal(reservation.ID, result.Namespace)
	require.Equal("2001:cdba::3257:9652", result.IP)
	require.EqualValues(zdbPort, result.Port)
}
