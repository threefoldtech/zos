// Package graphql for grid graphql support
package graphql

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// Nodes from graphql
type Nodes struct {
	Nodes []Node `json:"nodes"`
}

// Node from graphql
type Node struct {
	NodeID       uint32       `json:"nodeID"`
	PublicConfig PublicConfig `json:"publicConfig"`
}

// PublicConfig includes the public config information for the node
type PublicConfig struct {
	Ipv4 string `json:"ipv4"`
	Ipv6 string `json:"ipv6"`
}

// ListPublicNodes returns a list of public nodes
func (g *GraphQl) ListPublicNodes(n int, ipv4, ipv6 bool) ([]Node, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	offset := r.Intn(n)

	pubCond := ""
	if ipv4 {
		pubCond = `ipv4_isNull: false, ipv4_not_eq: ""`
	}
	if ipv6 {
		pubCond += `, ipv6_isNull: false, ipv6_not_eq: ""`
	}

	options := fmt.Sprintf(`(limit: %d, where: { publicConfig: {%s} }, offset: %d)`, n, pubCond, offset)

	nodesData, err := g.Query(fmt.Sprintf(`query getNodes{
            nodes%s {
              publicConfig {
								ipv4
								ipv6
							}
							nodeID
            }
          }`, options),
		map[string]interface{}{
			"nodes": n,
		})

	if err != nil {
		return []Node{}, err
	}

	nodesJSONData, err := json.Marshal(nodesData)
	if err != nil {
		return []Node{}, err
	}

	var res Nodes
	err = json.Unmarshal(nodesJSONData, &res)
	if err != nil {
		return []Node{}, err
	}

	return res.Nodes, nil
}
