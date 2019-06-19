package veth

import (
	"github.com/vishvananda/netlink"
)

func New(name string) (*netlink.Veth, error) {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	veth := &netlink.Veth{
		LinkAttrs: attrs,
	}

	return veth, netlink.LinkAdd(veth)
}
