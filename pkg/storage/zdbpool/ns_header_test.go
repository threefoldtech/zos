package zdbpool

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadHeader(t *testing.T) {
	f, err := os.Open("./test_data/zdb-namespace")
	require.NoError(t, err)
	defer f.Close()

	h, err := ReadHeader(f)
	require.NoError(t, err)

	assert.Equal(t, "test", h.Name)
	assert.Equal(t, "", h.Password)
	assert.Equal(t, uint64(1025), h.MaxSize)
}

func TestReadHeaderExtended(t *testing.T) {
	f, err := os.Open("./test_data/zdb-namespace.v2")
	require.NoError(t, err)
	defer f.Close()

	h, err := ReadHeader(f)
	require.NoError(t, err)

	assert.Equal(t, "test", h.Name)
	assert.Equal(t, "azmy", h.Password)
	assert.Equal(t, uint64(0x40000000), h.MaxSize) // 1G
}
