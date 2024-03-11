package upgrade

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/0-fs/meta"
	"github.com/threefoldtech/zos/pkg/upgrade/hub"
)

func TestUpgraderDownload(t *testing.T) {
	require := require.New(t)

	hubClient := hub.NewHubClient(defaultHubTimeout)
	up := &Upgrader{
		root: "/tmp/zfs-test-cache",
		hub:  hubClient,
	}

	err := Storage(defaultHubStorage)(up)
	require.NoError(err)

	const repo = "azmy.3bot"
	const flist = "test-flist.flist"

	store, err := up.getFlist(repo, flist)
	require.NoError(err)
	tmp := t.TempDir()

	err = up.copyRecursive(store, tmp)
	require.NoError(err)

	// validation of download
	err = store.Walk("", func(path string, info meta.Meta) error {
		downloaded := filepath.Join(tmp, path)
		stat, err := os.Lstat(downloaded)
		require.NoError(err)
		require.Equal(info.IsDir(), stat.IsDir())
		if info.IsDir() {
			return nil
		}

		switch info.Info().Type {
		case meta.RegularType:
			require.Equal(info.Info().Size, uint64(stat.Size()))
		case meta.LinkType:
			require.Equal(os.ModeSymlink, stat.Mode()&os.ModeType)
		}

		return nil
	})

	require.NoError(err)
}
