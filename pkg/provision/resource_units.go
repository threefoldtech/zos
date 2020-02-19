package provision

import (
	"encoding/json"
)

func (s *FSStore) processResourceUnits(r *Reservation, addOrRemoveBool bool) error {
	switch r.Type {
	case VolumeReservation:
		return s.processVolume(r, addOrRemoveBool)
	case ContainerReservation:
		return s.processContainer(r, addOrRemoveBool)
	case ZDBReservation:
		return s.processZdb(r, addOrRemoveBool)
	case KubernetesReservation:
		return s.processKubernetes(r, addOrRemoveBool)
	}

	return nil
}

func (s *FSStore) processVolume(r *Reservation, inc bool) error {
	var volume Volume
	if err := json.Unmarshal(r.Data, &volume); err != nil {
		return err
	}
	var c Counter
	switch volume.Type {
	case SSDDiskType:
		// volume.size in MB, but sru is in GB
		c = &s.counters.sru
	case HDDDiskType:
		c = &s.counters.hru
	}

	if inc {
		c.Add(int64(volume.Size))

	} else {
		c.Remove(int64(volume.Size))
	}

	return nil
}

func (s *FSStore) processContainer(r *Reservation, inc bool) error {
	var contCap ContainerCapacity
	if err := json.Unmarshal(r.Data, &contCap); err != nil {
		return err
	}
	var cpuCounter Counter = &s.counters.cru
	var memoryCounter Counter = &s.counters.mru

	if inc {
		cpuCounter.Add(int64(contCap.CPU))
		memoryCounter.Add(int64(contCap.Memory))
	} else {
		cpuCounter.Remove(int64(contCap.CPU))
		memoryCounter.Remove(int64(contCap.Memory))
	}

	return nil
}

func (s *FSStore) processZdb(r *Reservation, inc bool) error {
	var zdbVolume ZDB
	if err := json.Unmarshal(r.Data, &zdbVolume); err != nil {
		return err
	}
	var volumeCounter Counter
	switch zdbVolume.DiskType {
	case "SSD":
		volumeCounter = &s.counters.sru
	case "HDD":
		volumeCounter = &s.counters.hru
	}

	if inc {
		volumeCounter.Add(int64(zdbVolume.Size))
	} else {
		volumeCounter.Remove(int64(zdbVolume.Size))
	}

	return nil
}

func (s *FSStore) processKubernetes(r *Reservation, inc bool) error {
	var k8s Kubernetes
	if err := json.Unmarshal(r.Data, &k8s); err != nil {
		return err
	}
	var k8sCPUCounter Counter = &s.counters.cru
	var k8sMemoryCounter Counter = &s.counters.mru
	var k8sSSDCounter Counter = &s.counters.sru
	switch k8s.Size {
	case 1:
		if inc {
			k8sCPUCounter.Add(1)
			k8sMemoryCounter.Add(2048)
			k8sSSDCounter.Add(50)
		} else {
			k8sCPUCounter.Remove(1)
			k8sMemoryCounter.Remove(2048)
			k8sSSDCounter.Remove(50)
		}
	case 2:
		if inc {
			k8sCPUCounter.Add(2)
			k8sMemoryCounter.Add(4096)
			k8sSSDCounter.Add(100)
		} else {
			k8sCPUCounter.Remove(2)
			k8sMemoryCounter.Remove(4096)
			k8sSSDCounter.Remove(100)
		}
	}

	return nil
}
