package versioned

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func ExampleNewReader() {
	latest := New(1, 2, 0, "")
	// 1- Open file contains data
	buf := bytes.NewBufferString(`"v1.0.1beta" "my data goes here"`)

	// 2- create versioned reader
	reader, err := NewReader(buf)
	if err != nil {
		// no version in data, take another action!
		panic(err)
	}

	fmt.Println("data version is:", reader.Version())
	dec := json.NewDecoder(reader)

	if reader.Version().Compare(latest) <= 0 {
		//data version is older than or equal latest
		var data string
		if err := dec.Decode(&data); err != nil {
			panic(err)
		}
		fmt.Println("data is:", "my data goes here")
	}

	// Output:
	// data version is: v1.0.1beta
	// data is: my data goes here
}
