package gedis

import (
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/gedis/types/directory"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/threefoldtech/zosv2/modules/schema"
	"github.com/vishvananda/netlink"
)

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net"

// 	"github.com/pkg/errors"
// 	"github.com/threefoldtech/zosv2/modules"
// 	"github.com/threefoldtech/zosv2/modules/network/types"
// )

// //
// // TNoDB Interface
// //

// type registerAllocationBody struct {
// 	FarmerID string `json:"farmer_id"`
// 	Alloc    string `json:"allocation"`
// }

// type registerAllocationResponse struct {
// 	Status  int    `json:"status"`
// 	Message string `json:"message"`
// }

// type requestAllocationBody struct {
// 	FarmID string `json:"farm_id,omitempty"`
// }

// type requestAllocationResponse struct {
// 	Alloc     string `json:"allocation"`
// 	FarmAlloc string `json:"farm_allocation"`
// }

// type configurePublicIfaceBody struct {
// 	NodeID string          `json:"node_id,omitempty"`
// 	Iface  string          `json:"iface"`
// 	IPs    []string        `json:"ips"`
// 	GWs    []string        `json:"gateways"`
// 	Type   types.IfaceType `json:"iface_type"`
// }

// type configurePublicIfaceResponse struct {
// 	Status  int    `json:"status"`
// 	Message string `json:"message"`
// }

// type readPubIfaceBody struct {
// 	PublicConfig *types.PubIface `json:"public_config"`
// }

// type selectExitNodeBody struct {
// 	NodeID string `json:"node_id,omitempty"`
// }

// type selectExitNodeResponse struct {
// 	Status  int    `json:"status"`
// 	Message string `json:"message"`
// }

// type getNetworksVersionBody struct {
// 	NodeID string `json:"node_id,omitempty"`
// }

// type getNetworksBody struct {
// 	NetID string `json:"net_id"`
// }

// //RegisterAllocation implements network.TNoDB interface//RegisterAllocation implements network.TNoDB interface
// func (g *Gedis) RegisterAllocation(farm modules.Identifier, allocation *net.IPNet) error {
// 	req := registerAllocationBody{
// 		FarmerID: farm.Identity(),
// 		Alloc:    allocation.String(),
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return err
// 	}

// 	resp, err := g.sendCommand("allocations", "register", b)
// 	if err != nil {
// 		return parseError(err)
// 	}

// 	r := &registerAllocationResponse{}
// 	if err := json.Unmarshal(resp, &r); err != nil {
// 		return err
// 	}

// 	if r.Status != 1 { // FIXME: set code
// 		fmt.Printf("%+v", string(r.Message))
// 		return fmt.Errorf("wrong response status code received: %v", r.Message)
// 	}

// 	return nil
// }

// //RequestAllocation implements network.TNoDB interface
// func (g *Gedis) RequestAllocation(farm modules.Identifier) (*net.IPNet, *net.IPNet, error) {
// 	req := requestAllocationBody{
// 		FarmID: farm.Identity(),
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	resp, err := g.sendCommand("allocations", "get", b)
// 	if err != nil {
// 		return nil, nil, parseError(err)
// 	}

// 	data := &requestAllocationResponse{}
// 	if err := json.Unmarshal(resp, &data); err != nil {
// 		return nil, nil, err
// 	}

// 	_, alloc, err := net.ParseCIDR(data.Alloc)
// 	if err != nil {
// 		return nil, nil, errors.Wrap(err, "failed to parse network allocation")
// 	}

// 	_, farmAlloc, err := net.ParseCIDR(data.FarmAlloc)
// 	if err != nil {
// 		return nil, nil, errors.Wrap(err, "failed to parse farm allocation")
// 	}

// 	return alloc, farmAlloc, nil

// }

func (g *Gedis) getLocalInterfaces() ([]directory.TfgridNodeIface1, error) {
	output := []directory.TfgridNodeIface1{}

	links, err := netlink.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list interfaces")
	}

	for _, link := range ifaceutil.LinkFilter(links, []string{"device", "bridge"}) {

		if link.Attrs().Flags&net.FlagUp == 0 || link.Attrs().Name == "lo" {
			//interface is not up
			continue
		}

		if !ifaceutil.IsVirtEth(link.Attrs().Name) && !ifaceutil.IsPluggedTimeout(link.Attrs().Name, time.Second*5) {
			continue
		}

		_, gw, err := ifaceutil.HasDefaultGW(link)
		if err != nil {
			return nil, err
		}

		// TODO: i am only registering IPv4 addresses because jumpscale schema iprange
		// does not support IPv6 yet.
		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return nil, err
		}

		info := directory.TfgridNodeIface1{
			Name:    link.Attrs().Name,
			Addrs:   make([]schema.IPRange, len(addrs)),
			Gateway: make([]net.IP, 0),
		}

		for i, addr := range addrs {
			info.Addrs[i] = schema.IPRange{*addr.IPNet}
		}

		if gw != nil {
			info.Gateway = append(info.Gateway, gw)
		}

		output = append(output, info)
	}

	return output, err
}

//PublishInterfaces implements network.TNoDB interface
func (g *Gedis) PublishInterfaces(local modules.Identifier) error {
	ifaces, err := g.getLocalInterfaces()
	if err != nil {
		return err
	}

	_, err = g.Send("nodes", "publish_interfaces", Args{
		"node_id": local.Identity(),
		"ifaces":  ifaces,
	})

	return err
}

//ReadPubIface gets public config of a node
func (g *Gedis) ReadPubIface(node modules.Identifier) (*types.PubIface, error) {
	object, err := g.GetNode(node)
	if err != nil {
		return nil, err
	}

	if object.PublicConfig == nil {
		return nil, fmt.Errorf("public config not set")
	}

	return object.PublicConfig, nil
}

// //ConfigurePublicIface implements network.TNoDB interface
// func (g *Gedis) ConfigurePublicIface(node modules.Identifier, ips []*net.IPNet, gws []net.IP, iface string) error {
// 	req := configurePublicIfaceBody{
// 		NodeID: node.Identity(),
// 		Iface:  iface,
// 		IPs:    make([]string, len(ips)),
// 		GWs:    make([]string, len(gws)),
// 		Type:   types.MacVlanIface, //TODO: allow to chose type of connection
// 	}

// 	for i := range ips {
// 		req.IPs[i] = ips[i].String()
// 		req.GWs[i] = gws[i].String()
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return err
// 	}

// 	resp, err := g.sendCommand("nodes", "configure_public", b)
// 	if err != nil {
// 		return parseError(err)
// 	}

// 	r := configurePublicIfaceResponse{}
// 	if err := json.Unmarshal(resp, &r); err != nil {
// 		return err
// 	}

// 	if r.Status != 0 { // FIXME: set code
// 		return fmt.Errorf("wrong response status received: %s", r.Message)
// 	}

// 	return nil
// }

// //SelectExitNode implements network.TNoDB interface
// func (g *Gedis) SelectExitNode(node modules.Identifier) error {
// 	req := selectExitNodeBody{
// 		NodeID: node.Identity(),
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return err
// 	}

// 	resp, err := g.sendCommand("nodes", "select_exit", b)
// 	if err != nil {
// 		return parseError(err)
// 	}

// 	r := &selectExitNodeResponse{}
// 	if err := json.Unmarshal(resp, &r); err != nil {
// 		return err
// 	}

// 	if r.Status != 0 { // FIXME: set code
// 		return fmt.Errorf("wrong response status received: %s", r.Message)
// 	}

// 	return nil
// }

// //ReadPubIface implements network.TNoDB interface
// func (g *Gedis) ReadPubIface(node modules.Identifier) (*types.PubIface, error) {
// 	req := getNodeBody{
// 		NodeID: node.Identity(),
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := g.sendCommand("nodes", "get", b)
// 	if err != nil {
// 		return nil, parseError(err)
// 	}

// 	iface := readPubIfaceBody{}
// 	if err := json.Unmarshal(resp, &iface); err != nil {
// 		return nil, err
// 	}

// 	return iface.PublicConfig, nil
// }

// //GetNetwork implements network.TNoDB interface
// func (g *Gedis) GetNetwork(netid modules.NetID) (*modules.Network, error) {
// 	req := getNetworksBody{
// 		NetID: string(netid),
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := g.sendCommand("network", "get", b)
// 	if err != nil {
// 		return nil, parseError(err)
// 	}

// 	network := &modules.Network{}
// 	if err := json.Unmarshal(resp, &network); err != nil {
// 		return nil, err
// 	}

// 	return network, nil

// }

// //GetNetworksVersion implements network.TNoDB interface
// func (g *Gedis) GetNetworksVersion(nodeID modules.Identifier) (map[modules.NetID]uint32, error) {
// 	req := getNetworksVersionBody{
// 		NodeID: nodeID.Identity(),
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := g.sendCommand("networkVersion", "get", b)
// 	if err != nil {
// 		return nil, parseError(err)
// 	}

// 	versions := make(map[modules.NetID]uint32)
// 	if err := json.Unmarshal(resp, &versions); err != nil {
// 		return nil, err
// 	}

// 	return versions, nil
// }
