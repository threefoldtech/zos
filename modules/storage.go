package modules

import (
	"fmt"
)

// RaidProfile type
type RaidProfile string

const (
	// Single profile
	Single RaidProfile = "single"
	// Raid1 profile
	Raid1 RaidProfile = "raid1"
	// Raid10 profile
	Raid10 RaidProfile = "raid10"
)

// Validate make sure profile is correct
func (p RaidProfile) Validate() error {
	if _, ok := raidProfiles[p]; !ok {
		return fmt.Errorf("not supported raid profile '%s'", p)
	}

	return nil
}

var (
	raidProfiles = map[RaidProfile]struct{}{
		Single: struct{}{}, Raid1: struct{}{}, Raid10: struct{}{},
	}
	// DefaultPolicy value
	DefaultPolicy = StoragePolicy{
		Raid: Single,
	}

	// NullPolicy does not create pools
	NullPolicy = StoragePolicy{}
)

// StoragePolicy describes the pool creation policy
type StoragePolicy struct {
	// Raid profile for this policy
	Raid RaidProfile
	// Number of disks to use in a single pool
	// note that, the disks count must be valid for
	// the chosen raid profile.
	Disks uint8

	// Only create this amount of storage pools. Default to 0 -> unlimited.
	// The spared disks can later be used in automatic repair if a physical
	// disk got corrupt or bad.
	// Note that if it's set to 0 (unlimited), some disks might be spared anyway
	// in case the number of disks required in the policy doesn't add up to pools
	// for example, a pool of 2s on a machine with 5 disks.
	MaxPools uint8
}

// StorageModule defines the api for storage
type StorageModule interface {
	// Initialize, Scans available disks, then group them in `pools` to satisfy
	// the provided policy. For example, a Policy with `single` raid
	// profile, and 1 disk, will create a pool out of each free disk
	// on the node.
	Initialize(policy StoragePolicy) error
}
