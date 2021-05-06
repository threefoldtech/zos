package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// VMCustomSize type
type VMCustomSize struct {
	CRU int64   `json:"cru"`
	MRU float64 `json:"mru"`
	SRU float64 `json:"sru"`
}

func (p *Provisioner) vmDecomission(ctx context.Context, reservation *provision.Reservation) error {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		cfg VM
	)

	if err := json.Unmarshal(reservation.Data, &cfg); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	if err := vm.Delete(reservation.ID); err != nil {
		return errors.Wrapf(err, "failed to delete vm %s", reservation.ID)
	}

	netID := provision.NetworkID(reservation.User, string(cfg.NetworkID))
	if err := network.RemoveTap(netID); err != nil {
		return errors.Wrap(err, "could not clean up tap device")
	}

	if cfg.PublicIP != 0 {
		if err := network.RemovePubTap(pubIPResID(cfg.PublicIP)); err != nil {
			return errors.Wrap(err, "could not clean up public tap device")
		}
	}

	if err := storage.Deallocate(fmt.Sprintf("%s-%s", reservation.ID, "vda")); err != nil {
		return errors.Wrap(err, "could not remove vDisk")
	}

	return nil
}

func (p *Provisioner) buildNetworkInfo(ctx context.Context, rversion int, userID string, iface string, pubIface string, cfg VM) (pkg.VMNetworkInfo, error) {
	network := stubs.NewNetworkerStub(p.zbus)

	netID := provision.NetworkID(userID, string(cfg.NetworkID))
	subnet, err := network.GetSubnet(netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network resource subnet")
	}

	if !subnet.Contains(cfg.IP) {
		return pkg.VMNetworkInfo{}, fmt.Errorf("IP %s is not part of local nr subnet %s", cfg.IP.String(), subnet.String())
	}

	privNet, err := network.GetNet(netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network range")
	}

	addrCIDR := net.IPNet{
		IP:   cfg.IP,
		Mask: subnet.Mask,
	}

	gw4, gw6, err := network.GetDefaultGwIP(netID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrap(err, "could not get network resource default gateway")
	}

	privIP6, err := network.GetIPv6From4(netID, cfg.IP)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrap(err, "could not convert private ipv4 to ipv6")
	}

	networkInfo := pkg.VMNetworkInfo{
		Ifaces: []pkg.VMIface{{
			Tap:            iface,
			MAC:            "", // rely on static IP configuration so we don't care here
			IP4AddressCIDR: addrCIDR,
			IP4GatewayIP:   net.IP(gw4),
			IP4Net:         privNet,
			IP6AddressCIDR: privIP6,
			IP6GatewayIP:   gw6,
			Public:         false,
		}},
		Nameservers: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("1.1.1.1"), net.ParseIP("2001:4860:4860::8888")},
	}

	// from this reservation version on we deploy new VM's with the custom boot script for IP
	if rversion >= 2 {
		networkInfo.NewStyle = true
	}

	if cfg.PublicIP != 0 {
		// A public ip is set, load the reservation, extract the ip and make a config
		// for it

		pubIP, pubGw, err := p.getPubIPConfig(cfg.PublicIP)
		if err != nil {
			return pkg.VMNetworkInfo{}, errors.Wrap(err, "could not get public ip config")
		}

		// the mac address uses the global workload id
		// this needs to be the same as how we get it in the actual IP reservation
		mac := ifaceutil.HardwareAddrFromInputBytes([]byte(fmt.Sprintf("%d-1", pubIP)))

		iface := pkg.VMIface{
			Tap:            pubIface,
			MAC:            mac.String(), // mac so we always get the same IPv6 from slaac
			IP4AddressCIDR: pubIP,
			IP4GatewayIP:   pubGw,
			// for now we get ipv6 from slaac, so leave ipv6 stuffs this empty
			//
			Public: true,
		}

		networkInfo.Ifaces = append(networkInfo.Ifaces, iface)
	}

	return networkInfo, nil
}

// Get the public ip, and the gateway from the reservation ID
func (p *Provisioner) getPubIPConfig(rid schema.ID) (net.IPNet, net.IP, error) {
	// TODO: check if there is a better way to do this
	explorerClient, err := app.ExplorerClient()
	if err != nil {
		return net.IPNet{}, nil, errors.Wrap(err, "could not create explorer client")
	}

	// explorerClient.Workloads.Get(...) is currently broken
	workloadDefinition, err := explorerClient.Workloads.NodeWorkloadGet(fmt.Sprintf("%d-1", rid))
	if err != nil {
		return net.IPNet{}, nil, errors.Wrap(err, "could not load public ip reservation")
	}
	// load IP
	ip, ok := workloadDefinition.(*workloads.PublicIP)
	if !ok {
		return net.IPNet{}, nil, errors.Wrap(err, "could not decode ip reservation")
	}
	identity := stubs.NewIdentityManagerStub(p.zbus)
	self := identity.NodeID().Identity()
	selfDescription, err := explorerClient.Directory.NodeGet(self, false)
	if err != nil {
		return net.IPNet{}, nil, errors.Wrap(err, "could not get our own node description")
	}
	farm, err := explorerClient.Directory.FarmGet(schema.ID(selfDescription.FarmId))
	if err != nil {
		return net.IPNet{}, nil, errors.Wrap(err, "could not get our own farm")
	}

	var pubGw schema.IP
	for _, ips := range farm.IPAddresses {
		if ips.ReservationID == rid {
			pubGw = ips.Gateway
			break
		}
	}
	if pubGw.IP == nil {
		return net.IPNet{}, nil, errors.New("unable to identify public ip gateway")
	}

	return ip.IPaddress.IPNet, pubGw.IP, nil
}

// returns the vCpu's, memory, disksize for a vm size
// memory and disk size is expressed in MiB
func vmSize(vm VM) (cpu uint8, memory uint64, storage uint64, err error) {
	switch vm.Size {
	case -1:
		customSize := vm.Custom
		return uint8(customSize.CRU),
			uint64(customSize.MRU * 1024),
			uint64(customSize.SRU * 1024),
			nil
	case 1:
		// 1 vCpu, 2 GiB RAM, 50 GB disk
		return 1, 2 * 1024, 50 * 1024, nil
	case 2:
		// 2 vCpu, 4 GiB RAM, 100 GB disk
		return 2, 4 * 1024, 100 * 1024, nil
	case 3:
		// 2 vCpu, 8 GiB RAM, 25 GB disk
		return 2, 8 * 1024, 25 * 1024, nil
	case 4:
		// 2 vCpu, 8 GiB RAM, 50 GB disk
		return 2, 8 * 1024, 50 * 1024, nil
	case 5:
		// 2 vCpu, 8 GiB RAM, 200 GB disk
		return 2, 8 * 1024, 200 * 1024, nil
	case 6:
		// 4 vCpu, 16 GiB RAM, 50 GB disk
		return 4, 16 * 1024, 50 * 1024, nil
	case 7:
		// 4 vCpu, 16 GiB RAM, 100 GB disk
		return 4, 16 * 1024, 100 * 1024, nil
	case 8:
		// 4 vCpu, 16 GiB RAM, 400 GB disk
		return 4, 16 * 1024, 400 * 1024, nil
	case 9:
		// 8 vCpu, 32 GiB RAM, 100 GB disk
		return 8, 32 * 1024, 100 * 1024, nil
	case 10:
		// 8 vCpu, 32 GiB RAM, 200 GB disk
		return 8, 32 * 1024, 200 * 1024, nil
	case 11:
		// 8 vCpu, 32 GiB RAM, 800 GB disk
		return 8, 32 * 1024, 800 * 1024, nil
	case 12:
		// 1 vCpu, 64 GiB RAM, 200 GB disk
		return 1, 64 * 1024, 200 * 1024, nil
	case 13:
		// 1 mvCpu, 64 GiB RAM, 400 GB disk
		return 1, 64 * 1024, 400 * 1024, nil
	case 14:
		//1 vCpu, 64 GiB RAM, 800 GB disk
		return 1, 64 * 1024, 800 * 1024, nil
	case 15:
		//1 vCpu, 2 GiB RAM, 25 GB disk
		return 1, 2 * 1024, 25 * 1024, nil
	case 16:
		//2 vCpu, 4 GiB RAM, 50 GB disk
		return 2, 4 * 1024, 50 * 1024, nil
	case 17:
		//4 vCpu, 8 GiB RAM, 50 GB disk
		return 4, 8 * 1024, 50 * 1024, nil
	case 18:
		//1 vCpu, 1 GiB RAM, 25 GB disk
		return 1, 1 * 1024, 25 * 1024, nil
	}

	return 0, 0, 0, fmt.Errorf("unsupported vm size %d, only size -1, and 1 to 18 are supported", vm.Size)
}

func pubIPResID(reservationID schema.ID) string {
	// TODO: should this change in the actual reservation?
	return fmt.Sprintf("%d-1", reservationID)
}
