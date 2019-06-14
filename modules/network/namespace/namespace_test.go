package namespace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vishvananda/netlink"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateNetNS(t *testing.T) {
	name := "testns"
	_, err := CreateNetNS(name)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(netNSPath, name))
	assert.NoError(t, err)

	out, err := exec.Command("ip", "netns").CombinedOutput()
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(out), name))

	err = DeleteNetNS(name)
	require.NoError(t, err)

	out, err = exec.Command("ip", "netns").CombinedOutput()
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(out), name))
}

func printIfaces(ifaces []netlink.Link) {
	for _, iface := range ifaces {
		fmt.Println(iface.Attrs().Name)
	}
}
func TestNamespace(t *testing.T) {
	ifaces, err := netlink.LinkList()
	ifacesNr := len(ifaces)
	assert.True(t, ifacesNr > 0)
	// fmt.Println("before")
	// printIfaces(ifaces)

	nsName := "testns"
	_, err = CreateNetNS(nsName)
	require.NoError(t, err)
	defer DeleteNetNS(nsName)

	nsCtx := NSContext{}
	err = nsCtx.Enter(nsName)
	require.NoError(t, err)

	ifaces, err = netlink.LinkList()
	// fmt.Println("in namespace")
	// printIfaces(ifaces)
	assert.True(t, len(ifaces) == 1)

	err = netlink.LinkAdd(&netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: "dummy1",
		},
	})
	require.NoError(t, err)

	ifaces, err = netlink.LinkList()
	assert.True(t, len(ifaces) == 2)
	// fmt.Println("in namespace after add")
	// printIfaces(ifaces)

	err = nsCtx.Exit()
	require.NoError(t, err)

	ifaces, err = netlink.LinkList()
	require.NoError(t, err)
	// fmt.Println("after exit")
	// printIfaces(ifaces)
	found := false
	for _, iface := range ifaces {
		if iface.Attrs().Name == "dummy1" {
			found = true
		}
	}
	assert.Equal(t, ifacesNr, len(ifaces))
	assert.False(t, found)
}

// func TestWG(t *testing.T) {
// 	client, err := wgctrl.New()
// 	require.NoError(t, err)

// 	devices, err := client.Devices()
// 	require.NoError(t, err)

// 	for _, device := range devices {
// 		fmt.Printf("%+v\n", device)
// 	}
// }

// func TestCreateNetNSMultiple(t *testing.T) {
// 	name := "testns"
// 	_, err := CreateNetNS(name)
// 	require.NoError(t, err)

// 	name2 := "testns2"
// 	_, err = CreateNetNS(name2)
// 	require.NoError(t, err)

// 	err = DeleteNetNS(name)
// 	require.NoError(t, err)

// 	err = DeleteNetNS(name2)
// 	require.NoError(t, err)
// }
