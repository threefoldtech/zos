package networkd

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

func homeExistInterface() (netlink.Link, error) {
	master, err := netlink.LinkByName(types.DefaultBridge)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get default bridge")
	}

	all, err := netlink.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list links")
	}

	for _, link := range all {
		if link.Type() == "device" && link.Attrs().MasterIndex == master.Attrs().Index {
			return link, nil
		}
	}

	return nil, fmt.Errorf("failed to find home network device")
}

func ensureHostFw(ctx context.Context) error {

	log.Info().Msg("ensuring existing host nft rules")

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c",
		`
nft 'add table inet filter'
nft 'add table arp filter'
nft 'add table bridge filter'

# duo to a bug we had we need to make sure those chains are
# deleted and then recreated later
nft 'delete chain inet filter input'
nft 'delete chain inet filter forward'
nft 'delete chain inet filter output'

nft 'delete chain bridge filter input'
nft 'delete chain bridge filter forward'
nft 'delete chain bridge filter output'

nft 'delete chain arp filter input'
nft 'delete chain arp filter output'

# recreate chains correctly
nft 'add chain inet filter input   { type filter hook input priority filter; policy accept; }'
nft 'add chain inet filter forward { type filter hook forward priority filter; policy accept; }'
nft 'add chain inet filter output  { type filter hook output priority filter; policy accept; }'
nft 'add chain inet filter prerouting  { type filter hook prerouting priority filter; policy accept; }'

nft 'add chain arp filter input  { type filter hook input priority filter; policy accept; }'
nft 'add chain arp filter output { type filter hook output priority filter; policy accept; }'

nft 'add chain bridge filter input   { type filter hook input priority filter; policy accept; }'
nft 'add chain bridge filter forward { type filter hook forward priority filter; policy accept; }'
nft 'add chain bridge filter prerouting { type filter hook prerouting priority filter; policy accept; }'
nft 'add chain bridge filter postrouting { type filter hook postrouting priority filter; policy accept; }'
nft 'add chain bridge filter output  { type filter hook output priority filter; policy accept; }'

nft 'flush chain bridge filter forward'
nft 'flush chain inet filter forward'
nft 'flush chain inet filter prerouting'

# drop smtp traffic for hidden nodes
nft 'add rule inet filter prerouting iifname "b-*" tcp dport 25 reject with icmp type admin-prohibited'

# prevent access to local network
nft 'add rule bridge filter output oif eth0 ether daddr != "f6:27:cc:5b:12:fb" drop'
`)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "could not set up host nft rules")
	}

	return nil
}
