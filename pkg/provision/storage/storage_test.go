package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

func TestStorageAdd(t *testing.T) {
	require := require.New(t)
	root, err := ioutil.TempDir("", "storage-")
	require.NoError(err)
	defer os.RemoveAll(root)

	store, err := NewFSStore(root)
	require.NoError(err)

	err = store.Add(gridtypes.Workload{
		ID:   "my-id",
		User: "my-user",
		Type: gridtypes.VolumeReservation,
	})

	require.NoError(err)
	stat, err := os.Lstat(filepath.Join(root, pathByID, "my-id"))
	require.NoError(err)
	require.True(stat.Mode().IsRegular())

	stat, err = os.Lstat(filepath.Join(root, pathByType, "volume", "my-id"))
	require.NoError(err)
	require.Equal(os.ModeSymlink, stat.Mode()&os.ModeSymlink)

	stat, err = os.Lstat(filepath.Join(root, pathByUser, "my-user", pathByType, "volume", "my-id"))
	require.NoError(err)
	require.Equal(os.ModeSymlink, stat.Mode()&os.ModeSymlink)

	stat, err = os.Lstat(filepath.Join(root, pathByUser, "my-user", pathByID, "my-id"))
	require.NoError(err)
	require.Equal(os.ModeSymlink, stat.Mode()&os.ModeSymlink)

}

func TestStorageSet(t *testing.T) {
	require := require.New(t)
	root, err := ioutil.TempDir("", "storage-")
	require.NoError(err)
	defer os.RemoveAll(root)

	store, err := NewFSStore(root)
	require.NoError(err)

	err = store.Set(gridtypes.Workload{
		ID:   "my-id",
		User: "my-user",
		Type: gridtypes.VolumeReservation,
	})

	require.Error(err)
	require.True(errors.Is(err, provision.ErrWorkloadNotExists))

	err = store.Add(gridtypes.Workload{
		ID:   "my-id",
		User: "my-user",
		Type: gridtypes.VolumeReservation,
	})

	require.NoError(err)

	err = store.Set(gridtypes.Workload{
		ID:   "my-id",
		User: "my-user",
		Type: gridtypes.VolumeReservation,
	})

	require.NoError(err)
}

func TestStorageGet(t *testing.T) {
	require := require.New(t)
	root, err := ioutil.TempDir("", "storage-")
	require.NoError(err)
	fmt.Println(root)
	//defer os.RemoveAll(root)

	store, err := NewFSStore(root)
	require.NoError(err)

	wl := gridtypes.Workload{
		ID:   "my-id",
		User: "my-user",
		Type: gridtypes.VolumeReservation,
		Data: json.RawMessage(`"hello world"`),
	}

	err = store.Add(wl)
	require.NoError(err)

	loaded, err := store.Get(wl.ID)
	require.NoError(err)
	require.Equal(wl.Data, loaded.Data)
	require.Equal(wl.ID, loaded.ID)
	require.Equal(wl.User, loaded.User)
	require.Equal(wl.Type, loaded.Type)
}
