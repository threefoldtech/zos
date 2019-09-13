package gedis

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules/network"
	schema "github.com/threefoldtech/zosv2/modules/schema"

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
	Node TfgridNode2 `json:"node"`
}

//
// generated with schemac from model
//
type TfgridLocation1 struct {
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Continent string  `json:"continent"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type TfgridNodeResource1 struct {
	Cru int64 `json:"cru"`
	Mru int64 `json:"mru"`
	Hru int64 `json:"hru"`
	Sru int64 `json:"sru"`
}

type getNodeBody struct {
	NodeID string `json:"node_id,omitempty"`
}

type gedisListNodeBodyPayload struct {
	FarmID  string `json:"farmer_id"`
	Country string `json:"country"`
	City    string `json:"city"`
	Cru     int    `json:",omitempty"`
	Mru     int    `json:",omitempty"`
	Sru     int    `json:",omitempty"`
	Hru     int    `json:",omitempty"`
}

type gedisListNodeResponseBody struct {
	Nodes []TfgridNode2 `json:"nodes"`
}

type TfgridNodePublicIface1TypeEnum uint8

const (
	TfgridNodePublicIface1Type TfgridNodePublicIface1TypeEnum = iota
)

type TfgridNodeIface1 struct {
	Name    string           `json:"name"`
	Addrs   []schema.IPRange `json:"addrs"`
	Gateway []net.IP         `json:"gateway"`
}

type TfgridNodePublicIface1 struct {
	Master  string                         `json:"master"`
	Type    TfgridNodePublicIface1TypeEnum `json:"type"`
	Ipv4    net.IP                         `json:"ipv4"`
	Ipv6    net.IP                         `json:"ipv6"`
	Gw4     net.IP                         `json:"gw4"`
	Gw6     net.IP                         `json:"gw6"`
	Version int64                          `json:"version"`
}

type TfgridFarm1 struct {
	ThreebotId      string              `json:"threebot_id"`
	IyoOrganization string              `json:"iyo_organization"`
	Name            string              `json:"name"`
	WalletAddresses []string            `json:"wallet_addresses"`
	Location        TfgridLocation1     `json:"location"`
	Vta             string              `json:"vta"`
	ResourcePrice   TfgridNodeResource1 `json:"resource_price"`
}

type TfgridNode2 struct {
	NodeID           string                 `json:"node_id,omitempty"`
	FarmID           string                 `json:"farmer_id,omitempty"`
	Version          string                 `json:"os_version"`
	Uptime           int64                  `json:"uptime"`
	Address          string                 `json:"address"`
	Location         TfgridLocation1        `json:"location"`
	TotalResource    TfgridNodeResource1    `json:"total_resource"`
	UsedResource     TfgridNodeResource1    `json:"used_resource"`
	ReservedResource TfgridNodeResource1    `json:"reserved_resource"`
	Ifaces           []TfgridNodeIface1     `json:"ifaces"`
	PublicConfig     TfgridNodePublicIface1 `json:"public_config"`
	ExitNode         bool                   `json:"exit_node"`
	Approved         bool                   `json:"approved"`
}

type gedisNodeUpdateCapacity struct {
	NodeID   string              `json:"node_id"`
	Resource TfgridNodeResource1 `json:"resource"`
}

//
// Farms
//

type registerFarmBody struct {
	Farm string `json:"farm_id,omitempty"`
	Name string `json:"name,omitempty"`
}

type gedisRegisterFarmBody struct {
	Farm gedisRegisterFarmBodyPayload `json:"farm"`
}

type gedisRegisterFarmBodyPayload struct {
	ThreebotID string   `json:"threebot_id,omitempty"`
	Name       string   `json:"name,omitempty"`
	Email      string   `json:"email,omitempty"`
	Wallet     []string `json:"wallet_addresses"`
}

type getFarmBody struct {
	FarmID string `json:"farm_id,omitempty"`
}

type gedisGetFarmBody struct {
	Farm TfgridFarm1 `json:"farm"`
}

type gedisUpdateFarmBody struct {
	FarmID string      `json:"farm_id"`
	Farm   TfgridFarm1 `json:"farm"`
}

type gedisListFarmBody struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

type gedisListFarmResponseBody struct {
	Farms []TfgridFarm1 `json:"farms"`
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
		Node: TfgridNode2{
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

	r := TfgridNode2{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}

	// no need to do data conversion here, returns the id

	return r.NodeID, nil
}

func (g *Gedis) ListNode(farmID modules.Identifier, country string, city string) error {
	req := gedisListNodeBodyPayload{
		FarmID:  farmID.Identity(),
		Country: country,
		City:    city,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := g.sendCommand("nodes", "list", b)
	if err != nil {
		return parseError(err)
	}

	nl := &gedisListNodeResponseBody{}
	if err := json.Unmarshal(resp, &nl); err != nil {
		return err
	}

	// FIXME: gateway to list of network.Node
	fmt.Println(nl)

	return nil
}

func (g *Gedis) RegisterFarm(farm modules.Identifier, name string, email string, wallet []string) (string, error) {
	req := gedisRegisterFarmBody{
		Farm: gedisRegisterFarmBodyPayload{
			ThreebotID: farm.Identity(),
			Name:       name,
			Email:      email,
			Wallet:     wallet,
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	resp, err := g.sendCommand("farms", "register", b)
	if err != nil {
		fmt.Println("nope")
		return "", parseError(err)
	}

	fmt.Println(resp)

	r := TfgridFarm1{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}

	return r.Name, parseError(err)
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

	n := TfgridNode2{}
	if err := json.Unmarshal(resp, &n); err != nil {
		return nil, err
	}
	return &network.Node{
		NodeID: n.NodeID,
		FarmID: n.FarmID,
	}, nil
}

func (g *Gedis) updateGenericNodeCapacity(captype string, node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
	req := gedisNodeUpdateCapacity{
		NodeID: node.Identity(),
		Resource: TfgridNodeResource1{
			Mru: mru,
			Cru: cru,
			Hru: hru,
			Sru: sru,
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = g.sendCommand("nodes", "update_"+captype+"_capacity", b)
	if err != nil {
		return parseError(err)
	}

	return nil
}

func (g *Gedis) UpdateTotalNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
	return g.updateGenericNodeCapacity("total", node, mru, cru, hru, sru)
}

func (g *Gedis) UpdateReservedNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
	return g.updateGenericNodeCapacity("reserved", node, mru, cru, hru, sru)
}

func (g *Gedis) UpdateUsedNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
	return g.updateGenericNodeCapacity("used", node, mru, cru, hru, sru)
}

func (g *Gedis) GetFarm(farm modules.Identifier) (*network.Farm, error) {
	req := gedisGetFarmBody{
		Farm: TfgridFarm1{
			Name: farm.Identity(),
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(b))

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

func (g *Gedis) UpdateFarm(farm modules.Identifier) error {
	req := gedisUpdateFarmBody{
		FarmID: farm.Identity(),
		Farm: TfgridFarm1{
			ThreebotId:      "",
			IyoOrganization: "",
			Name:            "",
			WalletAddresses: []string{},
			// Location: {},
			Vta: "",
			// ResourcePrice: {},
		},
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = g.sendCommand("farms", "update", b)
	if err != nil {
		return parseError(err)
	}

	return nil
}

func (g *Gedis) ListFarm(country string, city string) error {
	req := gedisListFarmBody{
		Country: country,
		City:    city,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := g.sendCommand("farms", "list", b)
	if err != nil {
		return parseError(err)
	}

	fl := &gedisListFarmResponseBody{}
	if err := json.Unmarshal(resp, &fl); err != nil {
		return err
	}

	// FIXME: gateway to list of network.Farm
	fmt.Println(fl)

	return nil
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
