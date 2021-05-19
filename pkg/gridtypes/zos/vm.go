package zos

import (
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// VirtualMachine reservation data
type VirtualMachine struct {
	Name     string   `json:"name"`
	Network  string   `json:"network"`
	IP       net.IP   `json:"ip"`
	SSHKeys  []string `json:"ssh_keys"`
	PublicIP string   `json:"public_ip"`
	Size     uint8    `json:"size"`
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

	if v.IP.To4() == nil && v.IP.To16() == nil {
		return errors.New("invalid IP")
	}
	if v.Size < 1 || v.Size > 18 {
		return errors.New("unsupported vm size %d, only size 1 to 18 are supported")
	}
	wl, err := getter.Get(v.PublicIP)
	if err != nil {
		return fmt.Errorf("public ip is not found")
	}

	if wl.Type != PublicIPType {
		return errors.Wrapf(err, "workload of name '%s' is not a public ip", v.PublicIP)
	}

	return nil
}

// Capacity implementation
func (v VirtualMachine) Capacity() (gridtypes.Capacity, error) {

	rsu, ok := vmSize[v.Size]
	if !ok {
		return gridtypes.Capacity{}, fmt.Errorf("VM size %d is not supported", v.Size)
	}
	return rsu, nil
}

// Challenge creates signature challenge
func (v VirtualMachine) Challenge(b io.Writer) error {
	if _, err := fmt.Fprintf(b, "%s", v.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(b, "%s", v.Network); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(b, "%s", v.PublicIP); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(b, "%s", v.IP.String()); err != nil {
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

	return nil
}

// VirtualMachineResult result returned by VM reservation
type VirtualMachineResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}
