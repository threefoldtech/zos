package provision

import (
	"encoding/json"
	"io/ioutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"

	"os"
	"testing"
)

func TestEngine(t *testing.T) {
	td, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer t.Cleanup(func() {
		os.RemoveAll(td)
	})

	nodeID := "BhPhHVhfU8qerzzh1BGBgcQ7SQxQtND3JwuxSPoRzqkY"
	store := &FSStore{
		root: td,
	}

	engine := &defaultEngine{
		nodeID: nodeID,
		store:  store,
	}

	mustJSONMarshal := func(v interface{}) []byte {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return b
	}

	err = engine.store.Add(&Reservation{
		ID:   "1-1",
		Type: VolumeReservation,
		Data: mustJSONMarshal(Volume{
			Size: 1,
			Type: HDDDiskType,
		}),
	})
	require.NoError(t, err)

	err = engine.store.Add(&Reservation{
		ID:   "3-1",
		Type: ZDBReservation,
		Data: mustJSONMarshal(ZDB{
			Size:     15,
			Mode:     pkg.ZDBModeSeq,
			DiskType: pkg.SSDDevice,
		}),
	})
	require.NoError(t, err)

	err = engine.store.Add(&Reservation{
		ID:   "4-1",
		Type: ContainerReservation,
		Data: mustJSONMarshal(Container{
			Capacity: ContainerCapacity{
				CPU:    2,
				Memory: 4096,
			},
		}),
	})
	require.NoError(t, err)

	amount := engine.resourceAmount()
	assert.Equal(t, int64(2), amount.Cru)
	assert.Equal(t, int64(4), amount.Mru)
	assert.Equal(t, int64(15), amount.Sru)
	assert.Equal(t, int64(1), amount.Hru)
}
