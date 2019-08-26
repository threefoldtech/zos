package versioned

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var (
	NotVersioned = fmt.Errorf("no version information")
)

// Reader is a versioned reader
// The Versioned Reader is a reader that can load the version of the data from a stream
// without assuming anything regarding the underlying encoding of your data object
type Reader struct {
	io.Reader
	version Version
}

// Version return the version of the data
func (r *Reader) Version() Version {
	return r.version
}

// NewReader creates a new versioned reader from a stream. It fails
// if the reader can not read the version from the stream.
// On success, the reader will have a version, and then can be used
// to load the data.
func NewReader(r io.Reader) (*Reader, error) {
	dec := json.NewDecoder(r)

	var version Version
	if err := dec.Decode(&version); err != nil {
		// TODO: make a differentiation between IO error
		// and invalid version error.
		return nil, err
	}

	return &Reader{
		Reader:  io.MultiReader(dec.Buffered(), r),
		version: version,
	}, nil
}

// NewWriter creates a versioned writer that marks data with the
// Provided version.
func NewWriter(w io.Writer, version Version) (io.Writer, error) {
	data, err := json.Marshal(version)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(data)
	return w, err
}

// ReadFile content
func ReadFile(path string) (Version, []byte, error) {
	all, err := ioutil.ReadFile(path)
	if err != nil {
		return MustParse("0.0.0"), nil, err
	}

	buf := bytes.NewBuffer(all)
	reader, err := NewReader(buf)
	if err != nil {
		return MustParse("0.0.0"), all, NotVersioned
	}
	data, err := ioutil.ReadAll(reader)
	return reader.Version(), data, err
}

// WriteFile versioned data to file
func WriteFile(filename string, version Version, data []byte, perm os.FileMode) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	defer file.Close()
	writer, err := NewWriter(file, version)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)

	return err
}
