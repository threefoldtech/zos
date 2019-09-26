package gedis

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/threefoldtech/zosv2/modules/provision"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules"
)

func TestPoll(t *testing.T) {
	gedis := testClient(t)

	reservations, err := gedis.Poll(modules.StrIdentifier("1"), true, time.Now())
	require.NoError(t, err)

	for _, r := range reservations {
		switch r.Type {
		case provision.VolumeReservation:
			var v provision.Volume
			err = json.Unmarshal(r.Data, &v)
			assert.NoError(t, err)

		case provision.ContainerReservation:
			var v provision.Container
			err = json.Unmarshal(r.Data, &v)
			assert.NoError(t, err)

		case provision.ZDBReservation:
			var v provision.ZDB
			err = json.Unmarshal(r.Data, &v)
			assert.NoError(t, err)

		case provision.NetworkReservation:
			var v provision.Network
			err = json.Unmarshal(r.Data, &v)
			assert.NoError(t, err)
		}
	}
}
