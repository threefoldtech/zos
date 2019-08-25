package versioned

import (
	"encoding/json"
	"fmt"
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

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		V1  string
		V2  string
		Out int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"v1.2.1", "v1.2.1", 0},
		{"v1.0.1", "v1.0.0", 1},
		{"v1.1.0", "v1.0.0", 1},
		{"v1.1.1", "v1.1.2", -1},
		{"v1.2.0", "v2.2.0", -1},
	}

	for _, testCase := range cases {
		t.Run(fmt.Sprintf("%s-%s", testCase.V1, testCase.V2), func(t *testing.T) {
			v1, err := Parse(testCase.V1)
			require.NoError(t, err)
			v2, err := Parse(testCase.V2)
			require.NoError(t, err)

			require.Equal(t, v1.Compare(v2), testCase.Out)
		})

	}
}

func TestVersionRange(t *testing.T) {
	r := NewRange(New(1, 0, 0, ""), New(1, 4, 0, ""))

	require.True(t, r.Has(New(1, 0, 0, "")))
	require.True(t, r.Has(New(1, 4, 0, "")))
	require.True(t, r.Has(New(1, 2, 0, "")))

	require.False(t, r.Has(New(1, 5, 0, "")))
	require.False(t, r.Has(New(0, 9, 0, "")))
}
