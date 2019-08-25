package versioned

import (
	"encoding/json"
	"io"
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
	enc := json.NewEncoder(w)
	return w, enc.Encode(version)
}
