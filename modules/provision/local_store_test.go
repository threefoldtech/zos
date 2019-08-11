package provision

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExpired(t *testing.T) {
	store := NewMemStore()

	reservations := []*Reservation{
		{
			ID:       "r-1",
			Duration: time.Second,
			Created:  time.Now().Add(-time.Second * 10),
		},
		{
			ID:       "r-2",
			Duration: time.Minute,
			Created:  time.Now(),
		},
	}

	for _, r := range reservations {
		err := store.Add(r)
		require.NoError(t, err)
	}

	expired, err := store.GetExpired()
	require.NoError(t, err)
	assert.Equal(t, 1, len(expired))
	assert.Equal(t, "r-1", expired[0].ID)
}
