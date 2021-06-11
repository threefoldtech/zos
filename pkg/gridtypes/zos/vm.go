package zos

import (
	"fmt"
	"io"
	"net"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

type MachineInterface struct {
	Network gridtypes.Name `json:"network"`
	IP      net.IP         `json:"ip"`
}
type MachineNetwork struct {
	PublicIP   gridtypes.Name     `json:"public_ip"`
	Interfaces []MachineInterface `json:"interfaces"`
}

func (n *MachineNetwork) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", n.PublicIP); err != nil {
		return err
	}

	for _, inf := range n.Interfaces {
		if _, err := fmt.Fprintf(w, "%s", inf.Network); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "%s", inf.IP.String()); err != nil {
			return err
		}
	}

	return nil
}

type MachineCapacity struct {
	CPU    uint8          `json:"cpu"`
	Memory gridtypes.Unit `json:"memory"`
}

func (c *MachineCapacity) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", c.CPU); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", c.Memory); err != nil {
		return err
	}

	return nil
}

type MachineMount struct {
	Name       gridtypes.Name `json:"name"`
	Mountpoint string         `json:"mountpoint"`
}

func (m *MachineMount) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", m.Name); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", m.Mountpoint); err != nil {
		return err
	}

	return nil
}

// VirtualMachine reservation data
type VirtualMachine struct {
	Name            string          `json:"name"`
	FList           string          `json:"flist"`
	Network         MachineNetwork  `json:"network"`
	SSHKeys         []string        `json:"ssh_keys"`
	Size            uint8           `json:"size"` // deprecated, use compute_capacity instead
	ComputeCapacity MachineCapacity `json:"compute_capacity"`
	Mounts          []Mount         `json:"mounts"`
	// following items are only available in container mode. if FList is for a container
	// not a VM.
	Entrypoint string            `json:"entrypoint"`
	Env        map[string]string `json:"env"`
}

// Valid implementation
func (v VirtualMachine) Valid(getter gridtypes.WorkloadGetter) error {
	if matched, _ := regexp.MatchString("^[0-9a-zA-Z-.]*$", v.Name); !matched {
		return errors.New("the name must consist of alphanumeric characters, dot, and dash ony")
	}

	for _, key := range v.SSHKeys {
		trimmed := strings.TrimSpace(key)
		if strings.ContainsAny(trimmed, "\t\r\n\f\"") {
			return errors.New("ssh keys can't contain intermediate whitespace chars or quotes other than white space")
		}
	}

	if len(v.Network.Interfaces) != 1 {
		return fmt.Errorf("only one network private network is supported at the moment")
	}

	for _, inf := range v.Network.Interfaces {
		if inf.IP.To4() == nil && inf.IP.To16() == nil {
			return errors.New("invalid IP")
		}
	}

	if v.Size < 1 || v.Size > 18 {
		return errors.New("unsupported vm size %d, only size 1 to 18 are supported")
	}
	if !v.Network.PublicIP.IsEmpty() {
		wl, err := getter.Get(v.Network.PublicIP)
		if err != nil {
			return fmt.Errorf("public ip is not found")
		}

		if wl.Type != PublicIPType {
			return errors.Wrapf(err, "workload of name '%s' is not a public ip", v.Network.PublicIP)
		}
	}

	return nil
}

// Capacity implementation
func (v VirtualMachine) Capacity() (gridtypes.Capacity, error) {
	if v.Size > 0 {
		rsu, ok := vmSize[v.Size]
		if !ok {
			return gridtypes.Capacity{}, fmt.Errorf("VM size %d is not supported", v.Size)
		}
		return rsu, nil
	}

	// otherwise
	return gridtypes.Capacity{
		CRU: uint64(v.ComputeCapacity.CPU),
		MRU: v.ComputeCapacity.Memory,
	}, nil
}

// Challenge creates signature challenge
func (v VirtualMachine) Challenge(b io.Writer) error {
	if _, err := fmt.Fprintf(b, "%s", v.Name); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%s", v.FList); err != nil {
		return err
	}

	if err := v.Network.Challenge(b); err != nil {
		return err
	}

	for _, key := range v.SSHKeys {
		if _, err := fmt.Fprintf(b, "%s", key); err != nil {
			return err
		}
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

	return nil
}

// VirtualMachineResult result returned by VM reservation
type VirtualMachineResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}
