package cache

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
)

func TestLocalStore(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(root)

	type fields struct {
		root string
	}
	type args struct {
		r *provision.Reservation
	}

	network := pkg.NetResource{
		Name: "tf_devnet",
	}

	json, _ := json.Marshal(network)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "main",
			fields: fields{
				root: root,
			},
			args: args{
				r: &provision.Reservation{
					ID:       "r-1",
					Created:  time.Now().UTC().Add(-time.Minute).Round(time.Second),
					Duration: time.Second * 10,
					Data:     json,
					Type:     "network",
					User:     "1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Fs{
				root: tt.fields.root,
			}

			err = s.Add(tt.args.r)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			actual, err := s.Get(tt.args.r.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.args.r.Duration, actual.Duration)
			assert.Equal(t, tt.args.r.Created, actual.Created)
			assert.Equal(t, tt.args.r.ID, actual.ID)

			_, err = s.Get("foo")
			require.Error(t, err)

			expired, err := s.GetExpired()
			require.NoError(t, err)
			assert.Equal(t, len(expired), 1)
			assert.Equal(t, tt.args.r.Duration, expired[0].Duration)
			assert.Equal(t, tt.args.r.Created, expired[0].Created)
			assert.Equal(t, tt.args.r.ID, expired[0].ID)

			err = s.Remove(actual.ID)
			assert.NoError(t, err)

			_, err = s.Get(tt.args.r.ID)
			require.Error(t, err)
		})
	}
}

func TestUpdateNetwork(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(root)

	s := &Fs{
		root: root,
	}

	r1 := &provision.Reservation{
		ID:      "r-1",
		Created: time.Now().UTC().Add(-time.Minute).Round(time.Second),
		Data: mustMarshal(pkg.NetResource{
			Name: "tf_devnet",
		}),
		Type: primitives.NetworkResourceReservation,
		User: "1",
	}

	r2 := &provision.Reservation{
		ID:      "r-2",
		Created: time.Now().UTC().Add(-time.Minute).Round(time.Second),
		Data: mustMarshal(pkg.NetResource{
			Name: "tf_devnet",
		}),
		Type: primitives.NetworkResourceReservation,
		User: "1",
	}

	// same network name but different user
	r3 := &provision.Reservation{
		ID:      "r-3",
		Created: time.Now().UTC().Add(-time.Minute).Round(time.Second),
		Data: mustMarshal(pkg.NetResource{
			Name: "tf_devnet",
		}),
		Type: primitives.NetworkResourceReservation,
		User: "2",
	}

	err = s.Add(r1)
	require.NoError(t, err)

	exists, err := s.Exists(r1.ID)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.Add(r2)
	require.NoError(t, err)

	exists, err = s.Exists(r1.ID)
	require.NoError(t, err)
	require.False(t, exists)

	exists, err = s.Exists(r2.ID)
	require.NoError(t, err)
	require.True(t, exists)

	err = s.Add(r3)
	require.NoError(t, err)

	exists, err = s.Exists(r2.ID)
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = s.Exists(r3.ID)
	require.NoError(t, err)
	require.True(t, exists)
}

func mustMarshal(s interface{}) json.RawMessage {
	json, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return json
}
