package gedis

import (
	"encoding/json"

	"github.com/threefoldtech/zosv2/modules/network"

	"github.com/garyburd/redigo/redis"
	"github.com/threefoldtech/zosv2/modules"
)

//
// IDStore Interface
//

type registerNodeBody struct {
	NodeID string `json:"node_id,omitempty"`
	FarmID string `json:"farm_id,omitempty"`
}

type registerFarmBody struct {
	Farm string `json:"farm_id,omitempty"`
	Name string `json:"name,omitempty"`
}

func (g *Gedis) sendCommand(actor string, method string, b []byte) ([]byte, error) {
	con := g.pool.Get()
	defer con.Close()

	resp, err := redis.Bytes(con.Do(g.cmd("nodes", "add"), b, g.headers))
	if err != nil {
		return []byte{}, err
	}

	return resp, nil
}

func (g *Gedis) RegisterNode(nodeID, farmID modules.Identifier) (string, error) {
	req := registerNodeBody{
		NodeID: nodeID.Identity(),
		FarmID: farmID.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	resp, err := g.sendCommand("nodes", "add", b)
	if err != nil {
		return "", err
	}

	r := registerNodeBody{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}
	return r.NodeID, nil
}

func (g *Gedis) RegisterFarm(farm modules.Identifier, name string) error {
	req := registerFarmBody{
		Farm: farm.Identity(),
		Name: name,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = g.sendCommand("farms", "add", b)
	return err
}

func (g *Gedis) GetNode(nodeID modules.Identifier) (*network.Node, error) {
	resp, err := g.sendCommand("nodes", "get", nil)
	if err != nil {
		return nil, err
	}

	n := network.Node{}
	if err := json.Unmarshal(resp, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

//
// TNoDB Interface
//

type registerAllocationBody struct {
	FarmerID string `json:"farmer_id"`
	Alloc    string `json:"allocation"`
}

type requestAllocationBody struct {
	Alloc     string `json:"allocation"`
	FarmAlloc string `json:"farm_allocation"`
}

type configurePublicIfaceBody struct {
	Iface string            `json:"iface"`
	IPs   []string          `json:"ips"`
	GWs   []string          `json:"gateways"`
	Type  network.IfaceType `json:"iface_type"`
}

type readPubIfaceBody struct {
	PublicConfig *network.PubIface `json:"public_config"`
}

/*
func (g *Gedis) RegisterAllocation(farm modules.Identifier, allocation *net.IPNet) error {

}

func (s *Gedis) RequestAllocation(farm modules.Identifier) (*net.IPNet, *net.IPNet, uint8, error) {

}

func (s *Gedis) GetFarm(farm modules.Identifier) (network.Farm, error) {

}

func (s *Gedis) GetNode(nodeID modules.Identifier) (*types.Node, error) {

}

func (s *Gedis) PublishInterfaces(local modules.Identifier) error {

}

func (s *Gedis) ConfigurePublicIface(node modules.Identifier, ips []*net.IPNet, gws []net.IP, iface string) error {

}

func (s *Gedis) SelectExitNode(node modules.Identifier) error {

}

func (s *Gedis) ReadPubIface(node modules.Identifier) (*types.PubIface, error) {

}

func (s *Gedis) GetNetwork(netid modules.NetID) (*modules.Network, error) {

}

func (s *Gedis) GetNetworksVersion(nodeID modules.Identifier) (map[modules.NetID]uint32, error) {

}
*/
