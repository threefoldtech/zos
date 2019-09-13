package gedis

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules/network"

	"github.com/garyburd/redigo/redis"
	"github.com/threefoldtech/zosv2/modules"
)

//
// IDStore Interface
//

// 'structNameWithoutPrefix' are internal struct used
// 'gedisStructNameBlabla' are gedis datatype wrappers

type registerNodeBody struct {
	NodeID  string `json:"node_id,omitempty"`
	FarmID  string `json:"farm_id,omitempty"`
	Version string `json:"os_version"`
}

type gedisRegisterNodeBody struct {
	Node gedisRegisterNodeBodyPayload `json:"node"`
}

type gedisRegisterNodeBodyPayload struct {
	NodeID  string `json:"node_id,omitempty"`
	FarmID  string `json:"farmer_id,omitempty"`
	Version string `json:"os_version"`
}

type registerFarmBody struct {
	Farm string `json:"farm_id,omitempty"`
	Name string `json:"name,omitempty"`
}

type gedisRegisterFarmBody struct {
	Farm gedisRegisterFarmBodyPayload `json:"farm"`
}

type gedisRegisterFarmBodyPayload struct {
	// Farm string `json:"threebot_id,omitempty"`
	Name string `json:"name,omitempty"`
}

type getNodeBody struct {
	NodeID string `json:"node_id,omitempty"`
}

type getFarmBody struct {
	FarmID string `json:"farm_id,omitempty"`
}

func (g *Gedis) sendCommand(actor string, method string, b []byte) ([]byte, error) {
	con := g.pool.Get()
	defer con.Close()

	resp, err := redis.Bytes(con.Do(g.cmd(actor, method), b, g.headers))
	if err != nil {
		return []byte{}, err
	}

	return resp, nil
}

func (g *Gedis) RegisterNode(nodeID, farmID modules.Identifier, version string) (string, error) {
	req := gedisRegisterNodeBody{
		Node: gedisRegisterNodeBodyPayload{
			NodeID:  nodeID.Identity(),
			FarmID:  farmID.Identity(),
			Version: version,
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	resp, err := g.sendCommand("nodes", "add", b)
	if err != nil {
		return "", parseError(err)
	}

	r := gedisRegisterNodeBodyPayload{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}

	// no need to do data conversion here, returns the id

	return r.NodeID, nil
}

func (g *Gedis) RegisterFarm(farm modules.Identifier, name string) error {
	req := gedisRegisterFarmBody{
		Farm: gedisRegisterFarmBodyPayload{
			// Farm: farm.Identity(),
			Name: name,
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = g.sendCommand("farms", "add", b)
	return parseError(err)
}

func (g *Gedis) GetNode(nodeID modules.Identifier) (*network.Node, error) {
	req := getNodeBody{
		NodeID: nodeID.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := g.sendCommand("nodes", "get", b)
	if err != nil {
		return nil, parseError(err)
	}

	n := gedisRegisterNodeBodyPayload{}
	if err := json.Unmarshal(resp, &n); err != nil {
		return nil, err
	}
	return &network.Node{
		NodeID: n.NodeID,
		FarmID: n.FarmID,
	}, nil
}

//
// TNoDB Interface
//

type registerAllocationBody struct {
	FarmerID string `json:"farmer_id"`
	Alloc    string `json:"allocation"`
}

type registerAllocationResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type requestAllocationBody struct {
	FarmID string `json:"farm_id,omitempty"`
}

type requestAllocationResponse struct {
	Alloc     string `json:"allocation"`
	FarmAlloc string `json:"farm_allocation"`
}

type configurePublicIfaceBody struct {
	NodeID string            `json:"node_id,omitempty"`
	Iface  string            `json:"iface"`
	IPs    []string          `json:"ips"`
	GWs    []string          `json:"gateways"`
	Type   network.IfaceType `json:"iface_type"`
}

type configurePublicIfaceResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type readPubIfaceBody struct {
	PublicConfig *network.PubIface `json:"public_config"`
}

type selectExitNodeBody struct {
	NodeID string `json:"node_id,omitempty"`
}

type selectExitNodeResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type getNetworksVersionBody struct {
	NodeID string `json:"node_id,omitempty"`
}

type getNetworksBody struct {
	NetID string `json:"net_id"`
}

func (g *Gedis) RegisterAllocation(farm modules.Identifier, allocation *net.IPNet) error {
	req := registerAllocationBody{
		FarmerID: farm.Identity(),
		Alloc:    allocation.String(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := g.sendCommand("allocations", "register", b)
	if err != nil {
		return parseError(err)
	}

	r := &registerAllocationResponse{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return err
	}

	if r.Status != 1 { // FIXME: set code
		fmt.Printf("%+v", string(r.Message))
		return fmt.Errorf("wrong response status code received: %v", r.Message)
	}

	return nil
}

func (g *Gedis) RequestAllocation(farm modules.Identifier) (*net.IPNet, *net.IPNet, error) {
	req := requestAllocationBody{
		FarmID: farm.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}

	resp, err := g.sendCommand("allocations", "get", b)
	if err != nil {
		return nil, nil, parseError(err)
	}

	data := &requestAllocationResponse{}
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, nil, err
	}

	_, alloc, err := net.ParseCIDR(data.Alloc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse network allocation")
	}

	_, farmAlloc, err := net.ParseCIDR(data.FarmAlloc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse farm allocation")
	}

	return alloc, farmAlloc, nil

}

func (g *Gedis) GetFarm(farm modules.Identifier) (*network.Farm, error) {
	req := getFarmBody{
		FarmID: farm.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := g.sendCommand("farms", "get", b)
	if err != nil {
		return nil, parseError(err)
	}

	f := &network.Farm{}
	if err := json.Unmarshal(resp, &farm); err != nil {
		return nil, err
	}

	return f, nil
}

/*
func (g *Gedis) PublishInterfaces(local modules.Identifier) error {

}
*/

func (g *Gedis) ConfigurePublicIface(node modules.Identifier, ips []*net.IPNet, gws []net.IP, iface string) error {
	req := configurePublicIfaceBody{
		NodeID: node.Identity(),
		Iface:  iface,
		IPs:    make([]string, len(ips)),
		GWs:    make([]string, len(gws)),
		Type:   network.MacVlanIface, //TODO: allow to chose type of connection
	}

	for i := range ips {
		req.IPs[i] = ips[i].String()
		req.GWs[i] = gws[i].String()
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := g.sendCommand("nodes", "configure_public", b)
	if err != nil {
		return parseError(err)
	}

	r := configurePublicIfaceResponse{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return err
	}

	if r.Status != 0 { // FIXME: set code
		return fmt.Errorf("wrong response status received: %s", r.Message)
	}

	return nil
}

func (g *Gedis) SelectExitNode(node modules.Identifier) error {
	req := selectExitNodeBody{
		NodeID: node.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := g.sendCommand("nodes", "select_exit", b)
	if err != nil {
		return parseError(err)
	}

	r := &selectExitNodeResponse{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return err
	}

	if r.Status != 0 { // FIXME: set code
		return fmt.Errorf("wrong response status received: %s", r.Message)
	}

	return nil
}

func (g *Gedis) ReadPubIface(node modules.Identifier) (*network.PubIface, error) {
	req := getNodeBody{
		NodeID: node.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := g.sendCommand("nodes", "get", b)
	if err != nil {
		return nil, parseError(err)
	}

	iface := readPubIfaceBody{}
	if err := json.Unmarshal(resp, &iface); err != nil {
		return nil, err
	}

	return iface.PublicConfig, nil
}

func (g *Gedis) GetNetwork(netid modules.NetID) (*modules.Network, error) {
	req := getNetworksBody{
		NetID: string(netid),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := g.sendCommand("network", "get", b)
	if err != nil {
		return nil, parseError(err)
	}

	network := &modules.Network{}
	if err := json.Unmarshal(resp, &network); err != nil {
		return nil, err
	}

	return network, nil

}

func (g *Gedis) GetNetworksVersion(nodeID modules.Identifier) (map[modules.NetID]uint32, error) {
	req := getNetworksVersionBody{
		NodeID: nodeID.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := g.sendCommand("networkVersion", "get", b)
	if err != nil {
		return nil, parseError(err)
	}

	versions := make(map[modules.NetID]uint32)
	if err := json.Unmarshal(resp, &versions); err != nil {
		return nil, err
	}

	return versions, nil
}
