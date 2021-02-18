package primitives

import (
	"encoding/json"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// AtomicValue value for safe increment/decrement
type AtomicValue uint64

// Increment counter atomically by one
func (c *AtomicValue) Increment(v uint64) uint64 {
	return atomic.AddUint64((*uint64)(c), v)
}

// Decrement counter atomically by one
func (c *AtomicValue) Decrement(v uint64) uint64 {
	// spinlock until the decrement succeeds
	for {
		current := c.Current()
		// make sure we don't decrement below 0
		dec := v
		if dec > current {
			dec = current
		}
		// compute new value
		n := current - dec
		// only swap if `current`, and therefore the above calculations,
		// are still valid
		if atomic.CompareAndSwapUint64((*uint64)(c), current, n) {
			return n
		}
	}
}

// Current returns the current value
func (c *AtomicValue) Current() uint64 {
	return atomic.LoadUint64((*uint64)(c))
}

// Counters tracks the amount of primitives workload deployed and
// the amount of resource unit used
type Counters struct {
	containers AtomicValue
	volumes    AtomicValue
	networks   AtomicValue
	zdbs       AtomicValue
	vms        AtomicValue

	SRU AtomicValue // SSD storage in bytes
	HRU AtomicValue // HDD storage in bytes
	MRU AtomicValue // Memory storage in bytes
	CRU AtomicValue // CPU count absolute
}

// // CurrentWorkloads return the number of each workloads provisioned on the system
// func (c *Counters) CurrentWorkloads() directory.WorkloadAmount {
// 	return directory.WorkloadAmount{
// 		Network:      uint16(c.networks.Current()),
// 		Volume:       uint16(c.volumes.Current()),
// 		ZDBNamespace: uint16(c.zdbs.Current()),
// 		Container:    uint16(c.containers.Current()),
// 		K8sVM:        uint16(c.vms.Current()),
// 	}
// }

// // CurrentUnits return the number of each resource units reserved on the system
// func (c *Counters) CurrentUnits() directory.ResourceAmount {
// 	gib := float64(gib)
// 	return directory.ResourceAmount{
// 		Cru: c.CRU.Current(),
// 		Mru: float64(c.MRU.Current()) / gib,
// 		Hru: float64(c.HRU.Current()) / gib,
// 		Sru: float64(c.SRU.Current()) / gib,
// 	}
// }

const (
	mib = uint64(1024 * 1024)
	gib = uint64(mib * 1024)
)

// Increment is called by the provision.Engine when a reservation has been provisionned
func (c *Counters) Increment(r *gridtypes.Workload) error {

	var (
		u   resourceUnits
		err error
	)

	switch r.Type {
	case zos.VolumeType:
		c.volumes.Increment(1)
		u, err = processVolume(r)
	case zos.ContainerType:
		c.containers.Increment(1)
		u, err = processContainer(r)
	case zos.ZDBType:
		c.zdbs.Increment(1)
		u, err = processZdb(r)
	case zos.KubernetesType:
		c.vms.Increment(1)
		u, err = processKubernetes(r)
	case zos.NetworkType:
		c.networks.Increment(1)
		u = resourceUnits{}
		err = nil

		//TODO: add case for ipv4
	default:
		u = resourceUnits{}
		err = nil
	}
	if err != nil {
		return err
	}

	c.CRU.Increment(u.CRU)
	c.MRU.Increment(u.MRU)
	c.SRU.Increment(u.SRU)
	c.HRU.Increment(u.HRU)

	return nil
}

// Decrement is called by the provision.Engine when a reservation has been decommissioned
func (c *Counters) Decrement(r *gridtypes.Workload) error {

	var (
		u   resourceUnits
		err error
	)

	switch r.Type {
	case zos.VolumeType:
		c.volumes.Decrement(1)
		u, err = processVolume(r)
	case zos.ContainerType:
		c.containers.Decrement(1)
		u, err = processContainer(r)
	case zos.ZDBType:
		c.zdbs.Decrement(1)
		u, err = processZdb(r)
	case zos.KubernetesType:
		c.vms.Decrement(1)
		u, err = processKubernetes(r)
	case zos.NetworkType:
		c.networks.Decrement(1)
		u = resourceUnits{}
		err = nil
		//TODO: add case for ipv4
	default:
		u = resourceUnits{}
		err = nil
	}
	if err != nil {
		return err
	}

	c.CRU.Decrement(u.CRU)
	c.MRU.Decrement(u.MRU)
	c.SRU.Decrement(u.SRU)
	c.HRU.Decrement(u.HRU)

	return nil
}

type resourceUnits struct {
	SRU uint64 `json:"sru,omitempty"`
	HRU uint64 `json:"hru,omitempty"`
	MRU uint64 `json:"mru,omitempty"`
	CRU uint64 `json:"cru,omitempty"`
}

// CheckMemoryRequirements checks memory requirements for a reservation and compares it to whats in the counters
// and what is the total memory on this node
func (c *Counters) CheckMemoryRequirements(r *gridtypes.Workload, totalMemAvailable uint64) error {
	var requestedUnits resourceUnits
	var err error

	switch r.Type {
	case zos.ContainerType:
		requestedUnits, err = processContainer(r)
		if err != nil {
			return err
		}

	case zos.KubernetesType:
		requestedUnits, err = processKubernetes(r)
		if err != nil {
			return err
		}
	}

	if requestedUnits.MRU != 0 {
		// If current MRU + requested MRU exceeds total, return an error
		if c.MRU.Current()+requestedUnits.MRU > totalMemAvailable {
			return errors.New("not enough free resources to support this memory size")
		}
	}

	return nil
}

func processVolume(r *gridtypes.Workload) (u resourceUnits, err error) {
	var volume zos.Volume
	if err = json.Unmarshal(r.Data, &volume); err != nil {
		return u, err
	}

	// volume.size and SRU is in GiB
	switch volume.Type {
	case zos.SSDDevice:
		u.SRU = volume.Size * gib
	case zos.HDDDevice:
		u.HRU = volume.Size * gib
	}

	return u, nil
}

func processContainer(r *gridtypes.Workload) (u resourceUnits, err error) {
	var cont zos.Container
	if err = json.Unmarshal(r.Data, &cont); err != nil {
		return u, err
	}
	u.CRU = uint64(cont.Capacity.CPU)
	// memory is in MiB
	u.MRU = cont.Capacity.Memory * mib
	if cont.Capacity.DiskType == zos.SSDDevice {
		u.SRU = cont.Capacity.DiskSize * mib
	} else if cont.Capacity.DiskType == zos.HDDDevice {
		u.HRU = cont.Capacity.DiskSize * mib
	}

	return u, nil
}

func processZdb(r *gridtypes.Workload) (u resourceUnits, err error) {

	var zdbVolume zos.ZDB
	if err := json.Unmarshal(r.Data, &zdbVolume); err != nil {
		return u, err
	}

	switch zdbVolume.DiskType {
	case zos.SSDDevice:
		u.SRU = zdbVolume.Size * gib
	case zos.HDDDevice:
		u.HRU = zdbVolume.Size * gib
	}

	return u, nil
}

func processKubernetes(r *gridtypes.Workload) (u resourceUnits, err error) {
	var k8s Kubernetes
	if err = json.Unmarshal(r.Data, &k8s); err != nil {
		return u, err
	}

	// size are defined at https://github.com/threefoldtech/zos/blob/master/pkg/provision/kubernetes.go#L311
	switch k8s.Size {
	case 1:
		u.CRU = 1
		u.MRU = 2 * gib
		u.SRU = 50 * gib
	case 2:
		u.CRU = 2
		u.MRU = 4 * gib
		u.SRU = 100 * gib
	case 3:
		u.CRU = 2
		u.MRU = 8 * gib
		u.SRU = 25 * gib
	case 4:
		u.CRU = 2
		u.MRU = 8 * gib
		u.SRU = 50 * gib
	case 5:
		u.CRU = 2
		u.MRU = 8 * gib
		u.SRU = 200 * gib
	case 6:
		u.CRU = 4
		u.MRU = 16 * gib
		u.SRU = 50 * gib
	case 7:
		u.CRU = 4
		u.MRU = 16 * gib
		u.SRU = 100 * gib
	case 8:
		u.CRU = 4
		u.MRU = 16 * gib
		u.SRU = 400 * gib
	case 9:
		u.CRU = 8
		u.MRU = 32 * gib
		u.SRU = 100 * gib
	case 10:
		u.CRU = 8
		u.MRU = 32 * gib
		u.SRU = 200 * gib
	case 11:
		u.CRU = 8
		u.MRU = 32 * gib
		u.SRU = 800 * gib
	case 12:
		u.CRU = 1
		u.MRU = 64 * gib
		u.SRU = 200 * gib
	case 13:
		u.CRU = 1
		u.MRU = 64 * gib
		u.SRU = 400 * gib
	case 14:
		u.CRU = 1
		u.MRU = 64 * gib
		u.SRU = 800 * gib
	case 15:
		u.CRU = 1
		u.MRU = 2 * gib
		u.SRU = 25 * gib
	case 16:
		u.CRU = 2
		u.MRU = 4 * gib
		u.SRU = 50 * gib
	case 17:
		u.CRU = 4
		u.MRU = 8 * gib
		u.SRU = 50 * gib
	case 18:
		u.CRU = 1
		u.MRU = 1 * gib
		u.SRU = 25 * gib
	}

	return u, nil
}
