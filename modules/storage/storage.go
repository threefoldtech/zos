package storage

import "github.com/threefoldtech/zosv2/modules"

type storageModule struct {
}

// New create a new storage module service
func New() modules.StorageModule {
	return &storageModule{}
}

/**
Initialize, must be called at least onetime each boot.
What Initialize will do is the following:
 - Try to mount prepared pools (if they are not mounted already)
 - Scan free devices, apply the policy.
 - If new pools were created, the pool is going to be mounted automatically
**/
func (s *storageModule) Initialize(policy modules.StoragePolicy) error {
	return nil
}
