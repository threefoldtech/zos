package explorer

import (
	"errors"
	"net"

	"github.com/threefoldtech/rivine/modules"
	"github.com/threefoldtech/rivine/pkg/api"
	"github.com/threefoldtech/rivine/types"
)

var (
	// ErrNoHealthyExplorers is returned if all explorers in the group fail to respond in time
	ErrNoHealthyExplorers = errors.New("No explorer could statisfy the request")
)

// GroupedExplorer is a Backend which can call multiple explorers, calling another explorer if one is down
type GroupedExplorer struct {
	explorers []*Explorer
}

// NewGroupedExplorer creates a new GroupedExplorer from existing regular Explorers
func NewGroupedExplorer(explorers ...*Explorer) *GroupedExplorer {
	return &GroupedExplorer{explorers: explorers}
}

// CheckAddress returns all interesting transactions and blocks related to a given unlockhash
func (e *GroupedExplorer) CheckAddress(addr types.UnlockHash) ([]api.ExplorerBlock, []api.ExplorerTransaction, error) {
	for _, explorer := range e.explorers {
		blocks, transactions, err := explorer.CheckAddress(addr)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			continue
		}
		return blocks, transactions, err
	}
	return nil, nil, ErrNoHealthyExplorers
}

// CurrentHeight returns the current chain height
func (e *GroupedExplorer) CurrentHeight() (types.BlockHeight, error) {
	for _, explorer := range e.explorers {
		height, err := explorer.CurrentHeight()
		if err, ok := err.(net.Error); ok && err.Timeout() {
			continue
		}
		return height, err
	}
	return 0, ErrNoHealthyExplorers
}

// SendTxn sends a txn to the backend to ultimately include it in the transactionpool
func (e *GroupedExplorer) SendTxn(tx types.Transaction) (types.TransactionID, error) {
	for _, explorer := range e.explorers {
		txID, err := explorer.SendTxn(tx)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			continue
		}
		return txID, err
	}
	return types.TransactionID{}, ErrNoHealthyExplorers
}

// GetChainConstants gets the currently active chain constants for this backend
func (e *GroupedExplorer) GetChainConstants() (modules.DaemonConstants, error) {
	for _, explorer := range e.explorers {
		cts, err := explorer.GetChainConstants()
		if err, ok := err.(net.Error); ok && err.Timeout() {
			continue
		}
		return cts, err
	}
	return modules.DaemonConstants{}, ErrNoHealthyExplorers
}

func (e *GroupedExplorer) Get(endpoint string) error {
	var (
		nerr net.Error
		err  error
		ok   bool
	)
	for _, explorer := range e.explorers {
		err = explorer.Get(endpoint)
		if nerr, ok = err.(net.Error); ok && nerr.Timeout() {
			continue
		}
		return err
	}
	return ErrNoHealthyExplorers
}

func (e *GroupedExplorer) GetWithResponse(endpoint string, responseBody interface{}) error {
	var (
		nerr net.Error
		err  error
		ok   bool
	)
	for _, explorer := range e.explorers {
		err = explorer.GetWithResponse(endpoint, responseBody)
		if nerr, ok = err.(net.Error); ok && nerr.Timeout() {
			continue
		}
		return err
	}
	return ErrNoHealthyExplorers
}

func (e *GroupedExplorer) Post(endpoint, data string) error {
	var (
		nerr net.Error
		err  error
		ok   bool
	)
	for _, explorer := range e.explorers {
		err = explorer.Post(endpoint, data)
		if nerr, ok = err.(net.Error); ok && nerr.Timeout() {
			continue
		}
		return err
	}
	return ErrNoHealthyExplorers
}

func (e *GroupedExplorer) PostWithResponse(endpoint, data string, responseBody interface{}) error {
	var (
		nerr net.Error
		err  error
		ok   bool
	)
	for _, explorer := range e.explorers {
		err = explorer.PostWithResponse(endpoint, data, responseBody)
		if nerr, ok = err.(net.Error); ok && nerr.Timeout() {
			continue
		}
		return err
	}
	return ErrNoHealthyExplorers
}
