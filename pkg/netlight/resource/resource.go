package resource

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zos/pkg/netlight/bridge"
	"github.com/threefoldtech/zos/pkg/netlight/ifaceutil"
	"github.com/threefoldtech/zos/pkg/netlight/macvlan"
	"github.com/threefoldtech/zos/pkg/netlight/mycelium"
	"github.com/threefoldtech/zos/pkg/netlight/namespace"
	"github.com/threefoldtech/zos/pkg/netlight/options"
	"github.com/vishvananda/netlink"
)

type Resource struct {
	name string
}

// Create creates a network name space and wire it to the master bridge
func Create(name string, master *netlink.Bridge, ndmzIP *net.IPNet, seed []byte, privateNet *net.IPNet) error {
	privateNetBr := fmt.Sprintf("r%s", name)
	myBr := fmt.Sprintf("m%s", name)
	nsName := fmt.Sprintf("n%s", name)

	bridges := []string{myBr}
	if privateNet != nil {
		bridges = append(bridges, privateNetBr)
	}

	for _, name := range bridges {
		if !bridge.Exists(name) {
			if _, err := bridge.New(name); err != nil {
				return fmt.Errorf("couldn't create bridge %s: %w", name, err)
			}
		}
	}

	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		netNS, err = namespace.Create(nsName)
		if err != nil {
			return fmt.Errorf("failed to create namespace %s", err)
		}
	}

	defer netNS.Close()

	err = netNS.Do(func(_ ns.NetNS) error {
		if privateNet != nil {
			if !ifaceutil.Exists("private", netNS) {
				if _, err = macvlan.Create("private", privateNetBr, netNS); err != nil {
					return err
				}
			}
		}

		// create public interface and attach it to ndmz bridge
		if !ifaceutil.Exists("public", netNS) {
			if _, err = macvlan.Create("public", master.Name, netNS); err != nil {
				return err
			}
		}
		err = setLinkAddr("public", ndmzIP)
		if err != nil {
			return fmt.Errorf("couldn't set link addr for public interface in namespace %s: %w", nsName, err)
		}

		if !ifaceutil.Exists("mycelium", netNS) {
			if _, err = macvlan.Create("mycelium", myBr, netNS); err != nil {
				return err
			}
		}

		return nil
	})

	// setup mycelium
	ctx := context.Background()
	_, err = setupMycelium(ctx, nsName, myBr, seed)
	if err != nil {
		return fmt.Errorf("couldn't setup mycelium in namespace %s: %w", nsName, err)
	}
	return nil
}

func setLinkAddr(name string, ip *net.IPNet) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}

	if err := options.Set(name, options.IPv6Disable(false)); err != nil {
		return fmt.Errorf("failed to enable ip6 on interface %s: %w", name, err)
	}

	addr := netlink.Addr{
		IPNet: ip,
	}
	err = netlink.AddrAdd(link, &addr)
	if err != nil && !os.IsExist(err) {
		return err
	}

	return netlink.LinkSetUp(link)
}

func setupMycelium(ctx context.Context, nsName string, bridge string, seed []byte) (myc *mycelium.MyceliumServer, err error) {

	// myc, err = mycelium.EnsureMycelium(ctx, seed, myNs)
	// if err != nil {
	// 	return myc, errors.Wrap(err, "failed to start mycelium")
	// }

	return
}
