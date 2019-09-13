package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	_, err := NewManager()
	require.NoError(t, err)
}
