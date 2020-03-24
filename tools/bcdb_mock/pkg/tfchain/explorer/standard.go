package explorer

import (
	tfcli "github.com/threefoldfoundation/tfchain/extensions/tfchain/client"
	"github.com/threefoldtech/rivine/pkg/client"
)

// MainnetGroupedExplorer is a GroupedExplorer preconfigured for the official public testnet explorers
type MainnetGroupedExplorer struct {
	*GroupedExplorer
}

// NewMainnetGroupedExplorer creates a preconfigured grouped explorer for the public testnet nodes
func NewMainnetGroupedExplorer() *MainnetGroupedExplorer {
	mainnetUrls := []string{
		"https://explorer.threefoldtoken.com",
		"https://explorer2.threefoldtoken.com",
		"https://explorer3.threefoldtoken.com",
		"https://explorer4.threefoldtoken.com",
	}
	var explorers []*Explorer
	for _, url := range mainnetUrls {
		explorers = append(explorers, NewExplorer(url, "Rivine-Agent", ""))
	}
	explorer := &MainnetGroupedExplorer{NewGroupedExplorer(explorers...)}

	// register transactions for standard network of tfchain
	bc, err := client.NewBaseClient(explorer, nil)
	if err != nil {
		panic(err)
	}
	tfcli.RegisterStandardTransactions(bc)

	return explorer
}

// Name of the backend
func (te *MainnetGroupedExplorer) Name() string {
	return "standard"
}
