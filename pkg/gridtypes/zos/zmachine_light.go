package zos

import (
	"fmt"
	"io"
	"sort"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// MachineNetworkLight structure
type MachineNetworkLight struct {
	// Mycelium IP config, if planetary is true, but Mycelium is not set we fall back
	// to yggdrasil support. Otherwise (if mycelium is set) a mycelium ip is used instead.
	Mycelium *MyceliumIP `json:"mycelium,omitempty"`

	// Interfaces list of user znets to join
	Interfaces []MachineInterface `json:"interfaces"`
}

// Challenge builder
func (n *MachineNetworkLight) Challenge(w io.Writer) error {
	for _, inf := range n.Interfaces {
		if _, err := fmt.Fprintf(w, "%s", inf.Network); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "%s", inf.IP.String()); err != nil {
			return err
		}
	}

	if n.Mycelium != nil {
		if err := n.Mycelium.Challenge(w); err != nil {
			return err
		}
	}

	return nil
}

// ZMachineLight reservation data
type ZMachineLight struct {
	// Flist of the zmachine, must be a valid url to an flist.
	FList string `json:"flist"`
	// Network configuration for machine network
	Network MachineNetworkLight `json:"network"`
	// Size of zmachine disk
	Size gridtypes.Unit `json:"size"`
	// ComputeCapacity configuration for machine cpu+memory
	ComputeCapacity MachineCapacity `json:"compute_capacity"`
	// Mounts configure mounts/disks attachments to this machine
	Mounts []MachineMount `json:"mounts"`

	// following items are only available in container mode. if FList is for a container
	// not a VM.

	// Entrypoint entrypoint of the container, if not set the configured one from the flist
	// is going to be used
	Entrypoint string `json:"entrypoint"`
	// Env variables available for a container
	Env map[string]string `json:"env"`
	// Corex works in container mode which forces replace the
	// entrypoing of the container to use `corex`
	Corex bool `json:"corex"`

	// GPU attached to the VM
	// the list of the GPUs ids must:
	// - Exist, obviously
	// - Not used by other VMs
	// - Only possible on `dedicated` nodes
	GPU []GPU `json:"gpu,omitempty"`
}

func (m *ZMachineLight) MinRootSize() gridtypes.Unit {
	// sru = (cpu * mem_in_gb) / 8
	// each 1 SRU is 50GB of storage
	cu := gridtypes.Unit(m.ComputeCapacity.CPU) * m.ComputeCapacity.Memory / (8 * gridtypes.Gigabyte)

	if cu == 0 {
		return 500 * gridtypes.Megabyte
	}

	return 2 * gridtypes.Gigabyte
}

func (m *ZMachineLight) RootSize() gridtypes.Unit {
	min := m.MinRootSize()
	if m.Size > min {
		return m.Size
	}

	return min
}

// Valid implementation
func (v ZMachineLight) Valid(getter gridtypes.WorkloadGetter) error {
	if len(v.Network.Interfaces) != 1 {
		return fmt.Errorf("only one network private network is supported at the moment")
	}

	for _, inf := range v.Network.Interfaces {
		if inf.IP.To4() == nil && inf.IP.To16() == nil {
			return fmt.Errorf("invalid IP")
		}
	}
	if v.ComputeCapacity.CPU == 0 {
		return fmt.Errorf("cpu capacity can't be 0")
	}
	if v.ComputeCapacity.Memory < 250*gridtypes.Megabyte {
		return fmt.Errorf("mem capacity can't be less that 250M")
	}
	minRoot := v.MinRootSize()
	if v.Size != 0 && v.Size < minRoot {
		return fmt.Errorf("disk size can't be less that %d. Set to 0 for minimum", minRoot)
	}

	for _, ifc := range v.Network.Interfaces {
		if ifc.Network == "ygg" || ifc.Network == "pub" || ifc.Network == "mycelium" { //reserved temporary
			return fmt.Errorf("'%s' is reserved network name", ifc.Network)
		}
	}

	mycelium := v.Network.Mycelium
	if mycelium != nil {
		if len(mycelium.Seed) != MyceliumIPSeedLen {
			return fmt.Errorf("invalid mycelium seed length expected 6 bytes")
		}
	}

	return nil
}

// Capacity implementation
func (v ZMachineLight) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{
		CRU: uint64(v.ComputeCapacity.CPU),
		MRU: v.ComputeCapacity.Memory,
		SRU: v.RootSize(),
	}, nil
}

// Challenge creates signature challenge
func (v ZMachineLight) Challenge(b io.Writer) error {
	if _, err := fmt.Fprintf(b, "%s", v.FList); err != nil {
		return err
	}

	if err := v.Network.Challenge(b); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%d", v.Size); err != nil {
		return err
	}

	if err := v.ComputeCapacity.Challenge(b); err != nil {
		return err
	}

	for _, mnt := range v.Mounts {
		if err := mnt.Challenge(b); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(b, "%s", v.Entrypoint); err != nil {
		return err
	}

	encodeEnv := func(w io.Writer, env map[string]string) error {
		keys := make([]string, 0, len(env))
		for k := range env {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if _, err := fmt.Fprintf(w, "%s=%s", k, env[k]); err != nil {
				return err
			}
		}

		return nil
	}

	if err := encodeEnv(b, v.Env); err != nil {
		return err
	}

	for _, gpu := range v.GPU {
		if _, err := fmt.Fprintf(b, "%s", gpu); err != nil {
			return err
		}
	}

	return nil
}

// ZMachineLiteResult result returned by VM reservation
type ZMachineLiteResult struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	MyceliumIP string `json:"mycelium_ip"`
}
