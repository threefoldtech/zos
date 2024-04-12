package storage

import (
	"slices"

	"github.com/threefoldtech/zos/pkg/storage/filesystem"
)

// utils for pool ordering and presence

// Presence needed
type Presence func(s *Module) []filesystem.Pool

func poolCmp(a, b filesystem.Pool) int {
	_, errA := a.Mounted()
	_, errB := b.Mounted()

	// if the 2 pools has the same mount state, then both can
	// be 0
	if errA != nil && errB != nil || errA == nil && errB == nil {
		return 0
	} else if errA == nil {
		// mounted pool comes first
		return -1
	} else {
		return 1
	}
}

func SSD(s *Module) []filesystem.Pool {
	// we need to sort them by mount
	slices.SortFunc(s.ssds, poolCmp)
	return s.ssds
}

func HDD(s *Module) []filesystem.Pool {
	// we need to sort them by mount
	slices.SortFunc(s.hdds, poolCmp)
	return s.hdds
}

// get available pools in defined presence
func (s *Module) pools(presence ...Presence) []filesystem.Pool {
	var results []filesystem.Pool
	for _, filter := range presence {
		filtered := filter(s)
		results = append(results, filtered...)
	}

	return results
}
