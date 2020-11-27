package primitives

import "github.com/threefoldtech/zos/pkg/provision"

// FilesystemName return a string to be used as filesystem name from
// a reservation object
func FilesystemName(r provision.Reservation) string {
	return r.ID
}
