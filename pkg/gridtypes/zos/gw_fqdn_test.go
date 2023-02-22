package zos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBackend(t *testing.T) {
	require := require.New(t)

	t.Run("test ip:port cases", func(t *testing.T) {
		backend := Backend("1.1.1.1:10")
		_, err := backend.Parse(true)
		require.NoError(err)

		backend = Backend("1.1.1.1:10")
		_, err = backend.Parse(false)
		require.Error(err)

		backend = Backend("1.1.1.1")
		_, err = backend.Parse(true)
		require.Error(err)

		backend = Backend("1.1.1.1:port")
		_, err = backend.Parse(true)
		require.Error(err)

		backend = Backend("ip:10")
		_, err = backend.Parse(true)
		require.Error(err)

		backend = Backend("1.1.1.1:1000000")
		_, err = backend.Parse(true)
		require.Error(err)
	})

	t.Run("test http://ip:port cases", func(t *testing.T) {
		backend := Backend("http://1.1.1.1:10")
		_, err := backend.Parse(false)
		require.NoError(err)

		backend = Backend("http://1.1.1.1:10")
		_, err = backend.Parse(true)
		require.Error(err)

		backend = Backend("http://1.1.1.1:port")
		_, err = backend.Parse(false)
		require.Error(err)

		backend = Backend("http://ip:10")
		_, err = backend.Parse(false)
		require.Error(err)
	})

	t.Run("test http://ip cases", func(t *testing.T) {
		backend := Backend("http://1.1.1.1")
		_, err := backend.Parse(false)
		require.NoError(err)

		backend = Backend("http://1.1.1.1")
		_, err = backend.Parse(true)
		require.Error(err)

		backend = Backend("http://ip")
		_, err = backend.Parse(false)
		require.Error(err)
	})
}

func TestParseBackendIP6(t *testing.T) {
	require := require.New(t)

	t.Run("test ip:port cases", func(t *testing.T) {
		backend := Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		_, err := backend.Parse(true)
		require.NoError(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		_, err = backend.Parse(false)
		require.Error(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:port")
		_, err = backend.Parse(true)
		require.Error(err)

		backend = Backend("[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:1000000")
		_, err = backend.Parse(true)
		require.Error(err)
	})

	t.Run("test http://ip:port cases", func(t *testing.T) {
		backend := Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		_, err := backend.Parse(false)
		require.NoError(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:10")
		_, err = backend.Parse(true)
		require.Error(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]:port")
		_, err = backend.Parse(false)
		require.Error(err)
	})

	t.Run("test http://ip cases", func(t *testing.T) {
		backend := Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]")
		_, err := backend.Parse(false)
		require.NoError(err)

		backend = Backend("http://[2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF]")
		_, err = backend.Parse(true)
		require.Error(err)
	})
}
