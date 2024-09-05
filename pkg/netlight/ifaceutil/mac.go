package ifaceutil

import (
	"bytes"
	"crypto/md5"
	"net"

	"github.com/decred/base58"
)

// HardwareAddrFromInputBytes returns a deterministic hardware address
// for a given byte slice.
func HardwareAddrFromInputBytes(b []byte) net.HardwareAddr {
	var (
		offset int
		hs     [md5.Size]byte
		buf    []byte
		addr   net.HardwareAddr
	)
outerLoop:
	for {
		hs = md5.Sum(b[:])
		offset = 0
		buf = hs[offset : offset+hwMacAddress]
		buf[0] = (buf[0] | 2) & 0xfe // Set local bit, ensure unicast address
		addr = net.HardwareAddr(buf)
		for !isHardwareAddrInValidRange(addr) {
			if offset < hwMaxHashOffset {
				offset++
				buf = hs[offset : offset+hwMacAddress]
				buf[0] = (buf[0] | 2) & 0xfe // Set local bit, ensure unicast address
				addr = net.HardwareAddr(buf)
			} else {
				b = hs[:]
				continue outerLoop
			}
		}
		return addr
	}
}

func DeviceNameFromInputBytes(input []byte) string {
	// Unique generate a unique predetermined short name based
	// on that ID and input value n.
	// returns a max of 13 char str which is suitable
	// to be used as devices and tap devices names.

	h := md5.Sum(input)
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}

	return string(b)
}

func isHardwareAddrInValidRange(addr net.HardwareAddr) bool {
	// possible range 1
	if bytes.Compare(addr[hwMacAddress-len(macPUR1):], macPUR1[:]) <= 0 {
		return bytes.Compare(macPLR1[:], addr[:len(macPLR1)]) <= 0
	}
	// possible range 2
	if bytes.Compare(addr[hwMacAddress-len(macPUR2):], macPUR2[:]) <= 0 {
		return bytes.Compare(macPLR2[:], addr[:len(macPLR2)]) <= 0 &&
			!bytes.Equal(macPR2EL[:], addr[:len(macPR2EL)])
	}
	// possible (last) range 3
	return bytes.Compare(addr[hwMacAddress-len(macPUR3):], macPUR3[:]) <= 0 &&
		bytes.Compare(macPLR3[:], addr[:len(macPLR3)]) <= 0
}

const (
	hwMacAddress    = 6
	hwMaxHashOffset = md5.Size - hwMacAddress
)

var (
	// Possible Range 1: 00:03: to 00:51:ff:
	macPUR1 = [3]byte{0x00, 0x51, 0xff}
	macPLR1 = [2]byte{0x00, 0x03}

	// Possible Range 2: 00:54: to 90:00:ff:
	//    (except: 33:33:--:--:--:--)
	macPUR2  = [3]byte{0x90, 0x00, 0xff}
	macPR2EL = [2]byte{0x33, 0x33}
	macPLR2  = [2]byte{0x00, 0x54}

	// Possible Range 3: 90:02: to ff:ff:ff:ff:ff:fe
	macPUR3 = [6]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xfe}
	macPLR3 = [2]byte{0x90, 0x02}
)
