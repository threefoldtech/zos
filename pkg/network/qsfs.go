package network

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/nft"
)

var _nft = `
flush ruleset

table inet filter {

  chain base_checks {
    # allow established/related connections
    ct state {established, related} accept
    # early drop of invalid connections
    ct state invalid drop
  }

  chain input {
    type filter hook input priority 0; policy drop;
    jump base_checks
    # port for prometheus
    tcp dport 9100 iifname ygg0 accept
	# accept only locally generated packets
	meta iif lo ct state new accept
	ip6 nexthdr icmpv6 accept
  }

  chain forward {
    type filter hook forward priority 0; policy drop;
    # is there already an existing stream? (outgoing)
    jump base_checks
  }
}

`

func applyQSFSFirewall(netns string) error {
	if err := nft.Apply(strings.NewReader(_nft), netns); err != nil {
		return errors.Wrap(err, "failed to apply nft rule set")
	}

	return nil
}

func (n *networker) waitYggIPs(netns string) (string, error) {
	var yggNet = net.IPNet{
		IP:   net.ParseIP("200::"),
		Mask: net.CIDRMask(7, 128),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	isYgg := func(ip net.IP) bool {
		return yggNet.Contains(ip)
	}

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			ips, _, err := n.Addrs("ygg0", netns)
			if err != nil {
				return "", errors.Wrap(err, "failed to get ygg0 address")
			}
			for _, ip := range ips {
				if isYgg(ip) {
					return net.IP(ip).String(), nil
				}
			}
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for ygg ips timedout: context cancelled")
		}
	}
}

func (n networker) QSFSNamespace(id string) string {
	netId := "qsfs:" + id
	hw := ifaceutil.HardwareAddrFromInputBytes([]byte(netId))
	return qsfsNamespacePrefix + strings.Replace(hw.String(), ":", "", -1)
}

func (n networker) QSFSPrepare(id string) (string, string, error) {
	netId := "qsfs:" + id
	netNSName := n.QSFSNamespace(id)
	netNs, err := createNetNS(netNSName)
	if err != nil {
		return "", "", err
	}
	defer netNs.Close()
	if err := n.ndmz.AttachNR(netId, netNSName, n.ipamLeaseDir); err != nil {
		return "", "", errors.Wrap(err, "failed to prepare qsfs namespace")
	}

	if err := applyQSFSFirewall(netNSName); err != nil {
		return "", "", err
	}

	if n.ygg == nil {
		return netNSName, "", nil
	}
	err = n.attachYgg(id, netNs)
	if err != nil {
		return "", "", err
	}
	ip, err := n.waitYggIPs(netNSName)
	if err != nil {
		return "", "", err
	}

	return netNSName, ip, err
}

func (n networker) QSFSDestroy(id string) error {
	netId := "qsfs:" + id

	netNSName := n.QSFSNamespace(id)

	if err := n.ndmz.DetachNR(netId, n.ipamLeaseDir); err != nil {
		log.Err(err).Str("namespace", netNSName).Msg("failed to detach qsfs namespace from ndmz")
	}
	netNs, err := namespace.GetByName(netNSName)
	if err != nil {
		return errors.Wrap(err, "didn't find qsfs namespace")
	}
	defer netNs.Close()
	if err := n.detachYgg(id, netNs); err != nil {
		// log and continue cleaning up
		log.Error().Err(err).Msg("couldn't detach ygg interface")
	}
	return n.destroy(netNSName)
}
