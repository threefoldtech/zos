package tfchain

import (
	"github.com/threefoldtech/rivine/modules"
	"github.com/threefoldtech/rivine/pkg/api"
	"github.com/threefoldtech/rivine/types"
)

// Backend is the minimal interface required by a Wallet to have a view of the chain
type Backend interface {
	// CheckAddress returns all interesting transactions and blocks related to a given unlockhash
	CheckAddress(types.UnlockHash) ([]api.ExplorerBlock, []api.ExplorerTransaction, error)
	// CurrentHeight returns the current chain height
	CurrentHeight() (types.BlockHeight, error)
	// SendTxn sends a txn to the backend to ultimately include it in the transactionpool
	SendTxn(types.Transaction) (types.TransactionID, error)
	// GetChainConstants gets the currently active chain constants for this backend
	GetChainConstants() (modules.DaemonConstants, error)
}
