package graphql

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	graphql "github.com/hasura/go-graphql-client"
)

// GraphQl for tf graphql client
type GraphQl struct {
	client *graphql.Client
}

// NewGraphQl creates a new tf graphql client
func NewGraphQl(url string) GraphQl {
	client := graphql.NewClient(url, nil)
	return GraphQl{client: client}
}

// Node from graphql
type Node struct {
	NodeID       uint32       `graphql:"nodeID"`
	PublicConfig PublicConfig `graphql:"publicConfig"`
}

// PublicConfig includes the public config information for the node
type PublicConfig struct {
	Ipv4 string `graphql:"ipv4"`
	Ipv6 string `graphql:"ipv6"`
}

// ListPublicNodes returns a list of public nodes
// if nodesNum is given the query will use a limit and offset
// farmID id if not equal 0 will add a condition for it
// excludeFarmID if not equal 0 will add a condition ro exclude the farm ID
// ipv4 pool to set a condition for non empty ipv4
// ipv6 pool to set a condition for non empty ipv6
func (g *GraphQl) ListPublicNodes(ctx context.Context, nodesNum int, farmID, excludeFarmID uint32, ipv4, ipv6 bool) ([]Node, error) {
	var pubCond string
	if ipv4 {
		pubCond = `ipv4_isNull: false, ipv4_not_eq: ""`
	}
	if ipv6 {
		pubCond += `, ipv6_isNull: false, ipv6_not_eq: ""`
	}

	var farmCond string
	if farmID != 0 {
		farmCond = fmt.Sprintf(", farmID_eq: %d", farmID)
	}

	var excludeFarmCond string
	if excludeFarmID != 0 {
		excludeFarmCond = fmt.Sprintf(", farmID_not_eq: %d", excludeFarmID)
	}

	nodeUpReportInterval := time.Minute * 40
	nodeUpInterval := time.Now().Unix() - 2*int64(nodeUpReportInterval.Seconds())
	whereCond := fmt.Sprintf(`where: { updatedAt_gte: %d, AND: {power_isNull: true, OR: {power: {state_eq: Up, target_eq: Up}}}, publicConfig: {%s} %s %s }`, nodeUpInterval, pubCond, farmCond, excludeFarmCond)

	itemCount, err := g.getItemTotalCount(ctx, "nodes", whereCond)
	if err != nil {
		return []Node{}, err
	}

	var limit string
	if nodesNum != 0 && itemCount > nodesNum {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		offset := r.Intn(nodesNum)

		limit = fmt.Sprintf("limit: %d, offset: %d", nodesNum, offset)
	}

	options := fmt.Sprintf(`%s, %s`, limit, whereCond)
	query := fmt.Sprintf("query{nodes(%s){nodeID publicConfig {ipv4 ipv6}}}", options)

	res := struct {
		Nodes []Node
	}{}

	if err := g.client.Exec(ctx, query, &res, nil); err != nil {
		return []Node{}, err
	}

	return res.Nodes, nil
}

func (g *GraphQl) getItemTotalCount(ctx context.Context, itemName string, options string) (int, error) {
	query := fmt.Sprintf(`query { items: %sConnection(%s, orderBy: id_ASC) { count: totalCount } }`, itemName, options)

	res := struct {
		Items struct {
			Count int `graphql:"count"`
		}
	}{}

	if err := g.client.Exec(ctx, query, &res, nil); err != nil {
		return 0, err
	}

	return res.Items.Count, nil
}
