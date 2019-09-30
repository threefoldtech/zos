package gedis

import (
	"fmt"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/gedis/types/directory"
	"github.com/threefoldtech/zosv2/modules/network/types"
	"github.com/threefoldtech/zosv2/modules/schema"
)

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
