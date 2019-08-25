package versioned

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	// versioned object
	buf := bytes.NewBufferString(`"v1.2.0" {"name": "Test", "value": "success"}`)

	reader, err := NewReader(buf)
	require.NoError(t, err)

	if ok := assert.Equal(t, New(1, 2, 0, ""), reader.Version()); !ok {
		t.Fatal()
	}

	// now read the object itself
	var data struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	dec := json.NewDecoder(reader)
	err = dec.Decode(&data)
	require.NoError(t, err)

	if ok := assert.Equal(t, "Test", data.Name); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, "success", data.Value); !ok {
		t.Error()
	}
}

func TestReaderInvalid(t *testing.T) {
	// case 1, no version information in stream
	buf := bytes.NewBufferString(`{"name": "Test", "value": "success"}`)

	_, err := NewReader(buf)
	require.Error(t, err)

	// case 2, invalid version string
	buf = bytes.NewBufferString(`"abc" {"name": "Test", "value": "success"}`)

	_, err = NewReader(buf)
	require.Error(t, err)

	// case 3, empty input
	buf = bytes.NewBufferString("")
	_, err = NewReader(buf)
	require.Error(t, err)
}

func TestWriterReader(t *testing.T) {
	type Data struct {
		Name string
		Age  float64
		Tags []string
	}

	var buf bytes.Buffer

	writer, err := NewWriter(&buf, New(1, 0, 0, ""))
	require.NoError(t, err)

	// Note you can replace json here with any encoder u like
	enc := json.NewEncoder(writer)
	data := Data{"Test", 20.0, []string{"version", "check"}}
	require.NoError(t, enc.Encode(data))

	// loading
	reader, err := NewReader(&buf)
	require.NoError(t, err)

	if ok := assert.Equal(t, New(1, 0, 0, ""), reader.Version()); !ok {
		t.Fatal()
	}

	dec := json.NewDecoder(reader)
	var loaded Data

	require.NoError(t, dec.Decode(&loaded))
	if ok := assert.Equal(t, data, loaded); !ok {
		t.Fatal()
	}
}
