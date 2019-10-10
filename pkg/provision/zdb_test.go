package provision

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
)

func TestZDBProvisionNew(t *testing.T) {
	t.Skip("not implemented")
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

	client.On("Request", module, version, "Path", reservation.ID).
		Return("/some/path", nil)

	result, err := zdbProvision(ctx, &reservation)
	require.NoError(err)
	require.EqualValues(VolumeResult{"reservation-id"}, result)
}
