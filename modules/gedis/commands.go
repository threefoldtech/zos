package gedis

import (
	"encoding/json"

	"github.com/threefoldtech/zosv2/modules/network"

	"github.com/garyburd/redigo/redis"
	"github.com/threefoldtech/zosv2/modules"
)

func (g *Gedis) RegisterNode(nodeID, farmID modules.Identifier) (string, error) {
	type body struct {
		NodeID string `json:"node_id,omitempty"`
		FarmID string `json:"farm_id,omitempty"`
	}
	req := body{
		NodeID: nodeID.Identity(),
		FarmID: farmID.Identity(),
	}

	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	con := g.pool.Get()
	defer con.Close()
	resp, err := redis.Bytes(con.Do(g.cmd("nodes", "add"), b, g.headers))
	if err != nil {
		return "", err
	}
	r := body{}
	if err := json.Unmarshal(resp, &r); err != nil {
		return "", err
	}
	return r.NodeID, nil
}

func (g *Gedis) RegisterFarm(farm modules.Identifier, name string) error {
	type body struct {
		Farm string `json:"farm_id,omitempty"`
		Name string `json:"name,omitempty"`
	}
	req := body{
		Farm: farm.Identity(),
		Name: name,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	con := g.pool.Get()
	defer con.Close()
	_, err = redis.Bytes(con.Do(g.cmd("nodes", "add"), b, g.headers))
	return err
}

func (g *Gedis) GetNode(nodeID modules.Identifier) (*network.Node, error) {
	con := g.pool.Get()
	defer con.Close()

	resp, err := redis.Bytes(con.Do(g.cmd("nodes", "get"), "", g.headers))
	if err != nil {
		return nil, err
	}

	n := network.Node{}
	if err := json.Unmarshal(resp, &n); err != nil {
		return nil, err
	}
	return &n, nil
}
