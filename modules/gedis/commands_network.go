package gedis

import (
	"fmt"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/gedis/types/directory"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/threefoldtech/zosv2/modules/schema"
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

func (g *Gedis) getLocalInterfaces(ifaces []types.IfaceInfo) []directory.TfgridNodeIface1 {
	output := make([]directory.TfgridNodeIface1, 0, len(ifaces))
	for _, iface := range ifaces {
		output = append(output,
			directory.TfgridNodeIface1{
				Name: iface.Name,
				Addrs: func() []schema.IPRange {
					var o []schema.IPRange
					for _, r := range iface.Addrs {
						o = append(o, schema.IPRange{*r})
					}
					return o
				}(),
				Gateway: iface.Gateway,
			},
		)
	}

	return output
}

//PublishInterfaces implements network.TNoDB interface
func (g *Gedis) PublishInterfaces(local modules.Identifier, ifaces []types.IfaceInfo) error {
	output := g.getLocalInterfaces(ifaces)

	_, err := g.Send("nodes", "publish_interfaces", Args{
		"node_id": local.Identity(),
		"ifaces":  output,
	})

	return err
}

//GetPubIface gets public config of a node
func (g *Gedis) GetPubIface(node modules.Identifier) (*types.PubIface, error) {
	object, err := g.GetNode(node)
	if err != nil {
		return nil, err
	}

	if object.PublicConfig == nil {
		return nil, fmt.Errorf("public config not set")
	}

	return object.PublicConfig, nil
}

//SetPublicIface implements network.TNoDB interface
func (g *Gedis) SetPublicIface(node modules.Identifier, pub *types.PubIface) error {
	public := directory.TfgridNodePublicIface1{
		Master:  pub.Master,
		Type:    directory.TfgridNodePublicIface1TypeMacvlan,
		Version: int64(pub.Version),
	}

	if pub.IPv4 != nil {
		public.Ipv4 = schema.IPRange{*pub.IPv4}
		public.Gw4 = pub.GW4
	}

	if pub.IPv6 != nil {
		public.Ipv6 = schema.IPRange{*pub.IPv6}
		public.Gw6 = pub.GW6
	}

	_, err := g.Send("nodes", "set_public_iface", Args{
		"node_id": node.Identity(),
		"public":  public,
	})

	return err
}

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
