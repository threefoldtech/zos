package gedis

import (
	"testing"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClient(t *testing.T) *Gedis {
	gedis, err := New("tcp://172.17.0.2:8901", "default", "")
	require.NoError(t, err)

	err = gedis.Connect()
	require.NoError(t, err)
	return gedis
}

func TestPing(t *testing.T) {
	gedis := testClient(t)

	resp, err := gedis.Ping()
	require.NoError(t, err)
	assert.Equal(t, "PONG", resp)
}

func TestNodesCRUD(t *testing.T) {
	gedis := testClient(t)

	id, err := gedis.RegisterNode(modules.StrIdentifier("node1"), modules.StrIdentifier("farm1"), "2.0.0")
	require.NoError(t, err)

	node, err := gedis.GetNode(modules.StrIdentifier(id))
	require.NoError(t, err)

	assert.Equal(t, "node1", node.NodeID)
	assert.Equal(t, "farm1", node.FarmID)
}

// func TestListNode(t *testing.T) {
// 	gedis, err := New("tcp://172.17.0.2:8900", "default", "")
// 	require.NoError(t, err)

// 	err = gedis.Connect()
// 	require.NoError(t, err)

// 	nodes, err := gedis.ListNodes()
// 	require.NoError(t, err)
// 	fmt.Println(nodes)
// }

// func TestRegisterFarm(t *testing.T) {
// 	gedis, err := New("tcp://172.17.0.2:8900", "default", "")
// 	require.NoError(t, err)

// 	err = gedis.Connect()
// 	require.NoError(t, err)

// 	err = gedis.RegisterFarm(modules.StrIdentifier("farm1"), "my super farm")
// 	require.NoError(t, err)
// 	// id, err = gedis.RegisterNode(modules.StrIdentifier(fmt.Sprintf("%d", id)), modules.StrIdentifier("farm1"))
// 	// require.NoError(t, err)
// }
