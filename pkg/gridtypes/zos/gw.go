package zos

import (
	"fmt"
	"io"
	"net"
	"regexp"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	gwNameRegex = regexp.MustCompile(`^\w+$`)
)

type Backend string

func (b Backend) Valid() error {
	if _, err := net.ResolveTCPAddr("tcp", string(b)); err != nil {
		return errors.Wrap(err, "invalid backend address")
	}

	return nil
}

// GatewayProxy definition. this will proxy name.<zos.domain> to backends
type GatewayProxy struct {
	// Name of the domain prefix. this must be a valid dns name (with no dots)
	Name string `json:"name"`

	// Backends are list of backend ips
	Backends []Backend `json:"backends"`
}

func (g GatewayProxy) Valid(getter gridtypes.WorkloadGetter) error {
	if !gwNameRegex.MatchString(g.Name) {
		return fmt.Errorf("invalid name")
	}
	if len(g.Backends) == 0 {
		return fmt.Errorf("backends list can not be empty")
	}

	return nil
}

func (g GatewayProxy) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%s", g.Name); err != nil {
		return err
	}

	for _, backend := range g.Backends {
		if _, err := fmt.Fprintf(w, "%s", string(backend)); err != nil {
			return err
		}
	}

	return nil
}

func (g GatewayProxy) Capacity() (gridtypes.Capacity, error) {
	// this has to be calculated per bytes served over the gw. so
	// a special handler in reporting that need to calculate and report
	// this.
	return gridtypes.Capacity{}, nil
}

// GatewayProxyResult results
type GatewayProxyResult struct {
	FQDN string `json:"fqdn"`
}
