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
func (g *GraphQl) ListPublicNodes(n int, farmID uint32, ipv4, ipv6 bool) ([]Node, error) {
	var limit string
	if n != 0 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		offset := r.Intn(n)

		limit = fmt.Sprintf("limit: %d, offset: %d", n, offset)
	}

	var pubCond string
	if ipv4 {
		pubCond = `ipv4_isNull: false, ipv4_not_eq: ""`
	}
	if ipv6 {
		pubCond += `, ipv6_isNull: false, ipv6_not_eq: ""`
	}

	var farmCond string
	if farmID != 0 {
		farmCond = fmt.Sprintf("farmID_eq: %d", farmID)
	}

	options := fmt.Sprintf(`(%s, where: { publicConfig: {%s}, %s })`, limit, pubCond, farmCond)

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
