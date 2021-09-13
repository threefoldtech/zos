package flist

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListMounts(t *testing.T) {
	t.Skip("not implemented")

	require := require.New(t)
	cmder := &testCommander{T: t}
	strg := &StorageMock{}

	root, err := ioutil.TempDir("", "flist_root")
	require.NoError(err)
	defer os.RemoveAll(root)

	sys := &testSystem{}

	flister := newFlister(root, strg, cmder, sys)

	//TODO: finish test
	_, _ = flister.mounts(withParentDir("/"))

}
