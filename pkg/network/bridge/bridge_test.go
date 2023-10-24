package bridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
)

func TestCreateBridge(t *testing.T) {
	const ifName = "bro0"
	br, err := New(ifName)
	require.NoError(t, err)

	defer func() {
		_ = netlink.LinkDel(br)
	}()

	bridges, err := List()
	require.NoError(t, err)

	found := false
	for _, link := range bridges {
		if link.Type() == "bridge" && link.Attrs().Name == ifName {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestDeleteBridge(t *testing.T) {
	err := Delete("notexits")
	assert.NoError(t, err)

	const ifName = "bro0"
	br, err := New(ifName)
	require.NoError(t, err)

	// ensure bridge now exists
	link, err := netlink.LinkByName(br.LinkAttrs.Name)
	require.NoError(t, err)
	_, ok := link.(*netlink.Bridge)
	assert.True(t, ok)

	// delete it
	err = Delete(ifName)
	assert.NoError(t, err)

	// ensure bridge now it's gone
	_, err = netlink.LinkByName(br.LinkAttrs.Name)
	require.Error(t, err)
}

func TestAttachBridge(t *testing.T) {
	const ifName = "bro0"
	br, err := New(ifName)
	require.NoError(t, err)

	dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: "dummy0"}}
	err = netlink.LinkAdd(dummy)
	require.NoError(t, err)

	defer func() {
		_ = netlink.LinkDel(br)
		_ = netlink.LinkDel(dummy)
	}()

	err = attachNic(dummy, br, nil)
	assert.NoError(t, err)
}

func TestAttachBridgeWithMac(t *testing.T) {
	const ifName = "bro0"
	br, err := New(ifName)
	require.NoError(t, err)

	err = netlink.LinkAdd(&netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: "dummy0"}})
	require.NoError(t, err)

	dummy, err := netlink.LinkByName("dummy0")
	require.NoError(t, err)

	defer func() {
		_ = netlink.LinkDel(br)
		_ = netlink.LinkDel(dummy)
	}()

	addrBefore := br.Attrs().HardwareAddr
	err = AttachNicWithMac(dummy, br)
	require.NoError(t, err)

	l, err := netlink.LinkByName("bro0")
	require.NoError(t, err)
	br = l.(*netlink.Bridge)
	addrAfter := br.Attrs().HardwareAddr

	assert.NotEqual(t, addrBefore, addrAfter)
	assert.Equal(t, dummy.Attrs().HardwareAddr, addrAfter)
}
