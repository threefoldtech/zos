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

	h := Header{}
	ReadHeader(f, &h)
	require.NoError(t, err)

	assert.Equal(t, uint8(4), h.NameLength)
	assert.Equal(t, uint8(0), h.PasswordLength)
	assert.Equal(t, uint32(1025), h.MaxSize)
	assert.Equal(t, uint8(1), h.Flags)
}
