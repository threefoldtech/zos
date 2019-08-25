package versioned

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	version := New(1, 2, 3, "")

	if ok := assert.Equal(t, "v1.2.3", version.String()); !ok {
		t.Error()
	}

	data, err := version.MarshalText()
	require.NoError(t, err)

	if ok := assert.Equal(t, "v1.2.3", string(data)); !ok {
		t.Error()
	}
}

func TestParse(t *testing.T) {
	version, err := Parse("v2.3.4beta")

	require.NoError(t, err)

	if ok := assert.Equal(t, New(2, 3, 4, "beta"), version); !ok {
		t.Fatal()
	}
}

func TestJsonMarshal(t *testing.T) {
	v1 := New(1, 2, 3, "")
	v2 := New(2, 2, 3, "beta")

	object := struct {
		V1 Version  `json:"version"`
		V2 *Version `json:"another"`
	}{
		v1,
		&v2,
	}

	data, err := json.Marshal(object)
	require.NoError(t, err)

	if ok := assert.Equal(t, `{"version":"v1.2.3","another":"v2.2.3beta"}`, string(data)); !ok {
		t.Fatal()
	}
}

func TestJsonUnmarshal(t *testing.T) {
	var object struct {
		V1 Version  `json:"version"`
		V2 *Version `json:"another"`
	}

	err := json.Unmarshal([]byte(`{"version":"v1.2.3","another":"v2.2.3beta"}`), &object)
	require.NoError(t, err)

	if ok := assert.Equal(t, New(1, 2, 3, ""), object.V1); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, New(2, 2, 3, "beta"), *object.V2); !ok {
		t.Fatal()
	}
}
