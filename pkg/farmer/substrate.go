package farmer

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/substrate"
)

const (
	//DefaultFarmerTwinPort twin port
	DefaultFarmerTwinPort = uint16(3000)
)

// GetFarmTwin gets the IP of a farmer twin given the substrate db url
// and the farm id
func GetFarmTwin(url string, id uint32) (net.IP, error) {
	sub, err := substrate.NewSubstrate(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to substrate")
	}

	farm, err := sub.GetFarm(uint32(id))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get farm '%d'", id)
	}

	twin, err := sub.GetTwin(uint32(farm.TwinID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get twin")
	}

	ip := twin.IPAddress()
	if len(ip) == 0 {
		return nil, fmt.Errorf("invalid ip address associated with farmer twin")
	}

	return ip, nil
}

// NewClientFromSubstrate gets a farmer twin client from a substrate url and
// the farm id
func NewClientFromSubstrate(url string, id uint32, port ...uint16) (*Client, error) {
	ip, err := GetFarmTwin(url, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get farmer twin ip")
	}

	p := DefaultFarmerTwinPort
	if len(port) == 1 {
		p = port[0]
	} else if len(port) > 1 {
		return nil, fmt.Errorf("only one port is supported")
	}

	twinURL := fmt.Sprintf("http://[%s]:%d", ip.String(), p)

	return NewClient(twinURL)
}
