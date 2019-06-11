package network

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
)

// func TestInterfaces(t *testing.T) {

// 	links, err := interfaces()
// 	require.NoError(t, err)

// 	for _, link := range links {
// 		fmt.Printf("%v - %v", link.Attrs().Name, link.Type())
// 	}
// }

func TestCreateBridge(t *testing.T) {
	const ifName = "bro0"
	bridge, err := CreateBridge(ifName)
	require.NoError(t, err)

	defer func() {
		netlink.LinkDel(bridge)
	}()

	links, err := interfaces()
	require.NoError(t, err)
	found := false

	for _, link := range filterBridge(links) {
		fmt.Println(link.Attrs().Name)
		if link.Type() == "bridge" && link.Attrs().Name == ifName {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestAttachBridge(t *testing.T) {
	const ifName = "bro0"
	bridge, err := CreateBridge(ifName)
	require.NoError(t, err)

	defer func() {
		netlink.LinkDel(bridge)
	}()

	err = BridgeAttachNic(&netlink.Device{netlink.LinkAttrs{Name: "dummy0"}}, bridge)
	assert.NoError(t, err)
}
