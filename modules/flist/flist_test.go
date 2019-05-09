package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFlistModule(t *testing.T) (*flistModule, func()) {
	backend, err := ioutil.TempDir("", "backend")
	require.NoError(t, err)
	flist, err := ioutil.TempDir("", "flist")
	require.NoError(t, err)
	mountpoint, err := ioutil.TempDir("", "mounpoint")
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(backend)
		os.RemoveAll(mountpoint)
		os.RemoveAll(flist)
	}
	return New(backend, flist, mountpoint), cleanup
}
func TestMountUmount(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	f, cleanup := testFlistModule(t)
	defer cleanup()

	path, err := f.Mount("https://hub.grid.tf/zaibon/ubuntu_bionic.flist", "")
	require.NoError(err)

	assert.NotEqual("", path)
	infos, err := ioutil.ReadDir(path)
	require.NoError(err)
	assert.NotNil(infos)

	err = f.Umount(path)
	assert.NoError(err)
}
