package zdbpool

import (
	"encoding/binary"
	"io"
)

// Header is the structure contains information about a Namespace
type Header struct {
	NameLength     uint8  // length of the namespace name
	PasswordLength uint8  // length of the password
	MaxSize        uint32 // maximum datasize allowed on that namespace
	Flags          uint8
}

// ReadHeader parse the binary format of a header and fills the h object or return an error
func ReadHeader(r io.Reader, h *Header) error {
	return binary.Read(r, binary.LittleEndian, h)
}
