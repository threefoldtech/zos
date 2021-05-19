package zos

import (
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var k8sSize = map[uint8]gridtypes.Capacity{
	1: {
		CRU: 1,
		MRU: 2 * gridtypes.Gigabyte,
		SRU: 50 * gridtypes.Gigabyte,
	},
	2: {
		CRU: 2,
		MRU: 4 * gridtypes.Gigabyte,
		SRU: 100 * gridtypes.Gigabyte,
	},
	3: {
		CRU: 2,
		MRU: 8 * gridtypes.Gigabyte,
		SRU: 25 * gridtypes.Gigabyte,
	},
	4: {
		CRU: 2,
		MRU: 5 * gridtypes.Gigabyte,
		SRU: 50 * gridtypes.Gigabyte,
	},
	5: {
		CRU: 2,
		MRU: 8 * gridtypes.Gigabyte,
		SRU: 200 * gridtypes.Gigabyte,
	},
	6: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
		SRU: 50 * gridtypes.Gigabyte,
	},
	7: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
		SRU: 100 * gridtypes.Gigabyte,
	},
	8: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
		SRU: 400 * gridtypes.Gigabyte,
	},
	9: {
		CRU: 8,
		MRU: 32 * gridtypes.Gigabyte,
		SRU: 100 * gridtypes.Gigabyte,
	},
	10: {
		CRU: 8,
		MRU: 32 * gridtypes.Gigabyte,
		SRU: 200 * gridtypes.Gigabyte,
	},
	11: {
		CRU: 8,
		MRU: 32 * gridtypes.Gigabyte,
		SRU: 800 * gridtypes.Gigabyte,
	},
	12: {
		CRU: 1,
		MRU: 64 * gridtypes.Gigabyte,
		SRU: 200 * gridtypes.Gigabyte,
	},
	13: {
		CRU: 1,
		MRU: 64 * gridtypes.Gigabyte,
		SRU: 400 * gridtypes.Gigabyte,
	},
	14: {
		CRU: 1,
		MRU: 64 * gridtypes.Gigabyte,
		SRU: 800 * gridtypes.Gigabyte,
	},
	15: {
		CRU: 1,
		MRU: 2 * gridtypes.Gigabyte,
		SRU: 25 * gridtypes.Gigabyte,
	},
	16: {
		CRU: 2,
		MRU: 4 * gridtypes.Gigabyte,
		SRU: 50 * gridtypes.Gigabyte,
	},
	17: {
		CRU: 4,
		MRU: 8 * gridtypes.Gigabyte,
		SRU: 50 * gridtypes.Gigabyte,
	},
	18: {
		CRU: 1,
		MRU: 1 * gridtypes.Gigabyte,
		SRU: 25 * gridtypes.Gigabyte,
	},
}

// Kubernetes reservation data
type Kubernetes struct {
	// Size of the vm, this defines the amount of vCpu, memory, and the disk size
	// Docs: docs/kubernetes/sizes.md
	Size uint8 `json:"size"`
	// Network of the network namepsace in which to run the VM. The network
	// must be provisioned previously.
	Network string `json:"network"`
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IP `json:"ip"`
	// ClusterSecret is the hex encoded encrypted(?) cluster secret.
	ClusterSecret string `json:"cluster_secret"`
	// MasterIPs define the URL's for the kubernetes master nodes. If this
	// list is empty, this node is considered to be a master node.
	MasterIPs []net.IP `json:"master_ips"`
	// SSHKeys is a list of ssh keys to add to the VM. Keys can be either
	// a full ssh key, or in the form of `github:${username}`. In case of
	// the later, the VM will retrieve the github keys for this username
	// when it boots.
	SSHKeys []string `json:"ssh_keys"`
	// PublicIP points to a reservation for a public ip
	PublicIP string `json:"public_ip"`

	DatastoreEndpoint     string `json:"datastore_endpoint"`
	DisableDefaultIngress bool   `json:"disable_default_ingress"`
}

// Valid implementation
func (k Kubernetes) Valid(getter gridtypes.WorkloadGetter) error {
	wl, err := getter.Get(k.PublicIP)
	if err != nil {
		return fmt.Errorf("public ip is not found")
	}

	if wl.Type != PublicIPType {
		return errors.Wrapf(err, "workload of name '%s' is not a public ip", k.PublicIP)
	}

	return nil
}

// Challenge implementation
func (k Kubernetes) Challenge(b io.Writer) error {
	if _, err := fmt.Fprintf(b, "%d", k.Size); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(b, "%s", k.ClusterSecret); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(b, "%s", k.Network); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(b, "%s", k.IP.String()); err != nil {
		return err
	}
	for _, ip := range k.MasterIPs {
		if _, err := fmt.Fprintf(b, "%s", ip.String()); err != nil {
			return err
		}
	}
	for _, key := range k.SSHKeys {
		if _, err := fmt.Fprintf(b, "%s", key); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(b, "%s", k.PublicIP); err != nil {
		return err
	}

	return nil
}

// Capacity implementation
func (k Kubernetes) Capacity() (gridtypes.Capacity, error) {
	rsu, ok := k8sSize[k.Size]
	if !ok {
		return gridtypes.Capacity{}, fmt.Errorf("K8S VM size %d is not supported", k.Size)
	}

	return rsu, nil
}

// KubernetesResult result returned by k3s reservation
type KubernetesResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}