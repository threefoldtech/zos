package main

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/stubs"
)

func TestRPCMount(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	f, cleanup := testFlistModule(t)
	defer cleanup()

	server, err := zbus.NewRedisServer("flist", "tcp://localhost:6379", 1)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	server.Register(zbus.ObjectID{Name: "flist", Version: "0.0.1"}, f)
	go func() {
		err := server.Run(context.Background())
		require.NoError(err)
	}()

	client, err := zbus.NewRedisClient("tcp://localhost:6379")
	require.NoError(err)

	flist := stubs.NewFlisterStub(client)

	path, err := flist.Mount("https://hub.grid.tf/zaibon/ubuntu_bionic.flist", "")
	require.NoError(err)

	// check file are accessible
	assert.NotEqual("", path)
	infos, err := ioutil.ReadDir(path)
	require.NoError(err)
	assert.True(len(infos) > 0)

	// check pid file exists
	dir, name := filepath.Split(path)
	dir, _ = filepath.Split(dir[:len(dir)-1])
	pidPath := filepath.Join(dir, "pid", name) + ".pid"
	_, err = os.Stat(pidPath)
	assert.NoError(err)

	err = flist.Umount(path)
	assert.NoError(err)

	// check pid file is gone after unmount
	_, err = os.Stat(pidPath)
	assert.Error(err)
	assert.True(os.IsNotExist(err))
}
