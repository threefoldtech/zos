package network

import (
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
)

func CreateBridge(name string) (*netlink.Bridge, error) {
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
			// HardwareAddr: hw,
			TxQLen: 1000, //needed other wise bridge won't work
		},
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		return bridge, err
	}

	var err error
	defer func() {
		if err != nil {
			if err := netlink.LinkDel(bridge); err != nil {
				log.Error().Err(err).Msgf("failed to delete bridge %s", bridge.Name)
			}
		}
	}()

	return bridge, netlink.LinkSetUp(bridge)
}

func ListBridges() ([]*netlink.Bridge, error) {
	links, err := interfaces()
	if err != nil {
		return nil, err
	}
	return filterBridge(links), nil
}

func BridgeAttachNic(device *netlink.Device, bridge *netlink.Bridge) error {
	return netlink.LinkSetMaster(device, bridge)
}
