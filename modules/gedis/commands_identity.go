package gedis

import (
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/zosv2/modules/gedis/types/directory"

	"github.com/threefoldtech/zosv2/modules"
)

//
// IDStore Interface
//

//RegisterNode implements modules.IdentityManager interface
func (g *Gedis) RegisterNode(nodeID, farmID modules.Identifier, version string) (string, error) {
	resp, err := Bytes(g.Send("nodes", "add", Args{
		"node": directory.TfgridNode2{
			NodeId:    nodeID.Identity(),
			FarmId:    farmID.Identity(),
			OsVersion: version,
		},
	}))

	if err != nil {
		return "", parseError(err)
	}

	var out struct {
		Node directory.TfgridNode2 `json:"node"`
	}

	if err := json.Unmarshal(resp, &out); err != nil {
		return "", err
	}

	// no need to do data conversion here, returns the id
	return out.Node.NodeId, nil
}

// ListNode implements modules.IdentityManager interface
func (g *Gedis) ListNode(farmID modules.Identifier, country string, city string) ([]directory.TfgridNode2, error) {
	resp, err := Bytes(g.Send("nodes", "list", Args{
		"farm_id": farmID.Identity(),
		"country": country,
		"city":    city,
	}))

	if err != nil {
		return nil, err
	}

	fmt.Println(string(resp))
	var out struct {
		Nodes []directory.TfgridNode2 `json:"nodes"`
	}

	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}

	return out.Nodes, nil
}

//RegisterFarm implements modules.IdentityManager interface
func (g *Gedis) RegisterFarm(farm modules.Identifier, name string, email string, wallet []string) (int64, error) {
	resp, err := Bytes(g.Send("farms", "register", Args{
		"farm": directory.TfgridFarm1{
			ThreebotId:      farm.Identity(),
			Name:            name,
			Email:           email,
			WalletAddresses: wallet,
		},
	}))

	if err != nil {
		return 0, err
	}

	var out struct {
		FarmID json.Number `json:"farm_id"`
	}

	if err := json.Unmarshal(resp, &out); err != nil {
		return 0, err
	}

	return out.FarmID.Int64()
}

// //GetNode implements modules.IdentityManager interface
// func (g *Gedis) GetNode(nodeID modules.Identifier) (*types.Node, error) {
// 	req := getNodeBody{
// 		NodeID: nodeID.Identity(),
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := g.sendCommand("nodes", "get", b)
// 	if err != nil {
// 		return nil, parseError(err)
// 	}

// 	n := tfgridNode2{}
// 	if err := json.Unmarshal(resp, &n); err != nil {
// 		return nil, err
// 	}
// 	return &types.Node{
// 		NodeID: n.NodeID,
// 		FarmID: n.FarmID,
// 	}, nil
// }

// func (g *Gedis) updateGenericNodeCapacity(captype string, node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
// 	req := gedisNodeUpdateCapacity{
// 		NodeID: node.Identity(),
// 		Resource: tfgridNodeResource1{
// 			Mru: mru,
// 			Cru: cru,
// 			Hru: hru,
// 			Sru: sru,
// 		},
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = g.sendCommandBool("nodes", "update_"+captype+"_capacity", b)
// 	if err != nil {
// 		fmt.Println(err)
// 		return parseError(err)
// 	}

// 	return nil
// }

// //UpdateTotalNodeCapacity implements modules.IdentityManager interface
// func (g *Gedis) UpdateTotalNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
// 	return g.updateGenericNodeCapacity("total", node, mru, cru, hru, sru)
// }

// //UpdateReservedNodeCapacity implements modules.IdentityManager interface
// func (g *Gedis) UpdateReservedNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
// 	return g.updateGenericNodeCapacity("reserved", node, mru, cru, hru, sru)
// }

// //UpdateUsedNodeCapacity implements modules.IdentityManager interface
// func (g *Gedis) UpdateUsedNodeCapacity(node modules.Identifier, mru int64, cru int64, hru int64, sru int64) error {
// 	return g.updateGenericNodeCapacity("used", node, mru, cru, hru, sru)
// }

// //GetFarm implements modules.IdentityManager interface
// func (g *Gedis) GetFarm(farm modules.Identifier) (*network.Farm, error) {
// 	req := gedisGetFarmBody{
// 		Farm: tfgridFarm1{
// 			Name: farm.Identity(),
// 		},
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	fmt.Println(string(b))

// 	resp, err := g.sendCommand("farms", "get", b)
// 	if err != nil {
// 		return nil, parseError(err)
// 	}

// 	f := &network.Farm{}
// 	if err := json.Unmarshal(resp, &farm); err != nil {
// 		return nil, err
// 	}

// 	return f, nil
// }

// //UpdateFarm implements modules.IdentityManager interface
// func (g *Gedis) UpdateFarm(farm modules.Identifier) error {
// 	req := gedisUpdateFarmBody{
// 		FarmID: farm.Identity(),
// 		Farm: tfgridFarm1{
// 			ThreebotID:      "",
// 			IyoOrganization: "",
// 			Name:            "",
// 			WalletAddresses: []string{},
// 			// Location: {},
// 			Vta: "",
// 			// ResourcePrice: {},
// 		},
// 	}

// 	b, err := json.Marshal(req)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = g.sendCommand("farms", "update", b)
// 	if err != nil {
// 		return parseError(err)
// 	}

// 	return nil
// }

//ListFarm implements modules.IdentityManager interface
func (g *Gedis) ListFarm(country string, city string) ([]directory.TfgridFarm1, error) {
	result, err := Bytes(g.Send("farms", "list", Args{
		"country": country,
		"city":    city,
	}))

	if err != nil {
		return nil, err
	}

	var out struct {
		Farms []directory.TfgridFarm1 `json:"farms"`
	}

	err = json.Unmarshal(result, &out)
	return out.Farms, nil
}
