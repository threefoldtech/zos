package tfchain

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/rivine/crypto"
	"github.com/threefoldtech/rivine/modules"
	"github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/tfchain/explorer"
)

const (
	// maturityDelay
	maturityDelay = 720
	// arbitraryDataMaxSize is the maximum size of the arbitrary data field on a transaction
	arbitraryDataMaxSize = 83
)

var (
	// ErrWalletExists indicates that a wallet with that name allready exists when trying to create a new wallet
	ErrWalletExists = errors.New("A wallet with that name already exists")
	// ErrNoSuchWallet indicates that there is no wallet for a given name when trying to load a wallet
	ErrNoSuchWallet = errors.New("A wallet with that name does not exist")
	// ErrTooMuchData indicates that the there is too much data to add to the transction
	ErrTooMuchData = errors.New("Too much data is being supplied to the transaction")
	// ErrInsufficientWalletFunds indicates that the wallet does not have sufficient funds to fund the transaction
	ErrInsufficientWalletFunds = errors.New("Insufficient funds to create this transaction")
)

type (
	// Wallet represents a seed
	Wallet struct {
		seed modules.Seed
		keys map[types.UnlockHash]spendableKey
		// backend used to interact with the chain
		backend Backend

		index uint64
	}

	// spendableKey is the required information to spend an input associated with a key
	spendableKey struct {
		PublicKey crypto.PublicKey
		SecretKey crypto.SecretKey
	}

	// SpendableOutputs maps CoinOutputID's to their corresponding actual output
	SpendableOutputs map[types.CoinOutputID]types.CoinOutput
)

// NewWalletFromMnemonic creates a new wallet from a given mnemonic
func NewWalletFromMnemonic(mnemonic string, keysToLoad uint64, backendName string) (*Wallet, error) {
	seed, err := modules.InitialSeedFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}
	return NewWalletFromSeed(seed, keysToLoad, backendName)
}

// NewWalletFromSeed creates a new wallet with a given seed
func NewWalletFromSeed(seed modules.Seed, keysToLoad uint64, backendName string) (*Wallet, error) {
	backend, err := loadBackend(backendName)
	if err != nil {
		return nil, err
	}

	w := &Wallet{
		seed:    seed,
		backend: backend,
	}

	return w, nil
}

// LoadBackend loads a backend with the given name
func loadBackend(name string) (Backend, error) {
	switch name {
	case "standard":
		return explorer.NewMainnetGroupedExplorer(), nil
	case "testnet":
		return explorer.NewTestnetGroupedExplorer(), nil
	default:
		return nil, fmt.Errorf("No such network '%s'", name)
	}
}

// LoadAddresses loads addresses into wallet
// TODO IMPROVE! =]
func (w *Wallet) LoadAddresses(addresses []types.UnlockHash) error {
	missingAddresses := 0
	keyLength := len(addresses)
	startingIndex := w.index
	spendableKeys := make([]spendableKey, 0)
	for i := 0; i < keyLength; i++ {
		key, err := generateSpendableKey(w.seed, startingIndex+uint64(i))
		if err != nil {
			return err
		}
		spendableKeys = append(spendableKeys, key)
	}
	for _, key := range spendableKeys {
		uh, err := key.UnlockHash()
		if err != nil {
			return err
		}
		w.keys[uh] = key
		w.index += 1
	}
	for _, address := range addresses {
		if _, exists := w.keys[address]; !exists {
			missingAddresses += 1
		}
	}
	if missingAddresses != 0 {
		return errors.New("There are some missing addresses")
	}
	return nil
}

func generateSpendableKey(seed modules.Seed, index uint64) (spendableKey, error) {
	// Generate the keys and unlock conditions.
	entropy, err := crypto.HashAll(seed, index)
	if err != nil {
		return spendableKey{}, err
	}
	sk, pk := crypto.GenerateKeyPairDeterministic(entropy)
	return spendableKey{
		PublicKey: pk,
		SecretKey: sk,
	}, nil
}

// UnlockHash derives the unlockhash from the spendableKey
func (sk spendableKey) UnlockHash() (types.UnlockHash, error) {
	return types.NewEd25519PubKeyUnlockHash(sk.PublicKey)
}

func (w *Wallet) checkAddress(address types.UnlockHash, currentChainHeight types.BlockHeight) (SpendableOutputs, error) {
	blocks, transactions, err := w.backend.CheckAddress(address)
	if err != nil {
		return nil, err
	}
	tempMap := make(SpendableOutputs)

	// We scann the blocks here for the miner fees, and the transactions for actual transactions
	for _, block := range blocks {
		// Collect the miner fees
		// But only those that have matured already
		if block.Height+maturityDelay >= currentChainHeight {
			// ignore miner payout which hasn't yet matured
			continue
		}
		for i, minerPayout := range block.RawBlock.MinerPayouts {
			if minerPayout.UnlockHash == address {
				tempMap[block.MinerPayoutIDs[i]] = types.CoinOutput{
					Value: minerPayout.Value,
					Condition: types.UnlockConditionProxy{
						Condition: types.NewUnlockHashCondition(minerPayout.UnlockHash),
					},
				}
			}
		}
	}

	// Collect the transaction outputs
	for _, txn := range transactions {
		for i, utxo := range txn.RawTransaction.CoinOutputs {
			if utxo.Condition.UnlockHash() == address {
				tempMap[txn.CoinOutputIDs[i]] = utxo
			}
		}
	}
	// Remove the ones we've spent already
	for _, txn := range transactions {
		for _, ci := range txn.RawTransaction.CoinInputs {
			delete(tempMap, ci.ParentID)
		}
	}

	return tempMap, nil
}

// getBalance of an address
func (w *Wallet) getBalance(outputs SpendableOutputs) (types.Currency, error) {
	balance := types.NewCurrency64(0)

	for _, uco := range outputs {
		balance = balance.Add(uco.Value)
	}
	return balance, nil
}

// GetBalance of an address
func (w *Wallet) GetBalance(address types.UnlockHash) (types.Currency, error) {
	height, err := w.backend.CurrentHeight()
	if err != nil {
		return types.Currency{}, errors.Wrap(err, "Failed to get current height")
	}

	outputs, err := w.checkAddress(address, height)
	if err != nil {
		return types.Currency{}, errors.Wrap(err, "Failed to check address")
	}

	balance := types.NewCurrency64(0)
	for _, uco := range outputs {
		balance = balance.Add(uco.Value)
	}
	return balance, nil
}

// GenerateAddress generates an address
func (w *Wallet) GenerateAddress() (types.UnlockHash, error) {
	key, err := generateSpendableKey(w.seed, w.index)
	if err != nil {
		return types.UnlockHash{}, err
	}
	uh, err := key.UnlockHash()
	if err != nil {
		return types.UnlockHash{}, err
	}

	w.index = w.index + 1
	w.keys[uh] = key
	return uh, nil
}

// TransferCoins transfers coins by creating and submitting a V1 transaction.
// Data can optionally be included.
func (w *Wallet) TransferCoins(amount types.Currency, from types.UnlockHash, to types.UnlockHash, data []byte) (types.TransactionID, bool, error) {
	var fundsLeft bool
	// check data length
	if len(data) > arbitraryDataMaxSize {
		return types.TransactionID{}, false, ErrTooMuchData
	}

	chainCts, err := w.backend.GetChainConstants()
	if err != nil {
		return types.TransactionID{}, false, errors.Wrap(err, "Failed to get chainconstants")
	}

	height, err := w.backend.CurrentHeight()
	if err != nil {
		return types.TransactionID{}, false, errors.Wrap(err, "Failed to get current height")
	}

	outputs, err := w.checkAddress(from, height)
	if err != nil {
		return types.TransactionID{}, false, errors.Wrapf(err, "Failed to check address: %s", from.String())
	}

	walletBalance, err := w.GetBalance(from)
	if err != nil {
		return types.TransactionID{}, false, errors.Wrapf(err, "Failed to get balance: %s", from.String())
	}

	// we give only the minimum fee
	txFee := chainCts.MinimumTransactionFee

	// Since this is only for demonstration purposes, lets give a fixed 10 hastings fee
	// minerfee := types.NewCurrency64(10)

	// The total funds we will be spending in this transaction
	requiredFunds := amount.Add(txFee)

	// Verify that we actually have enough funds available in the wallet to complete the transaction
	if walletBalance.Cmp(requiredFunds) == -1 {
		return types.TransactionID{}, false, ErrInsufficientWalletFunds
	}

	// Create the transaction object
	var txn types.Transaction
	txn.Version = chainCts.DefaultTransactionVersion

	// Greedily add coin inputs until we have enough to fund the output and minerfee
	inputs := []types.CoinInput{}

	// Track the amount of coins we already added via the inputs
	inputValue := types.ZeroCurrency

	for id, utxo := range outputs {
		// If the inputValue is not smaller than the requiredFunds we added enough inputs to fund the transaction
		if inputValue.Cmp(requiredFunds) != -1 {
			break
		}
		// Append the input
		inputs = append(inputs, types.CoinInput{
			ParentID: id,
			Fulfillment: types.NewFulfillment(types.NewSingleSignatureFulfillment(
				types.Ed25519PublicKey(w.keys[utxo.Condition.UnlockHash()].PublicKey))),
		})
		// And update the value in the transaction
		inputValue = inputValue.Add(utxo.Value)
	}
	// Set the inputs
	txn.CoinInputs = inputs

	// sanity checking
	for _, inp := range inputs {
		if _, exists := w.keys[outputs[inp.ParentID].Condition.UnlockHash()]; !exists {
			return types.TransactionID{}, false, errors.New("Trying to spend unexisting output")
		}
	}

	txn.CoinOutputs = append(txn.CoinOutputs, types.CoinOutput{
		Value:     amount,
		Condition: types.UnlockConditionProxy{Condition: types.NewUnlockHashCondition(to)},
	})

	// So now we have enough inputs to fund everything. But we might have overshot it a little bit, so lets check that
	// and add a new output to ourself if required to consume the leftover value
	remainder := inputValue.Sub(requiredFunds)
	if !remainder.IsZero() {
		fundsLeft = true
		outputToSelf := types.CoinOutput{
			Value:     remainder,
			Condition: types.NewCondition(types.NewUnlockHashCondition(from)),
		}
		// add our self referencing output to the transaction
		txn.CoinOutputs = append(txn.CoinOutputs, outputToSelf)
	}

	// Add the miner fee to the transaction
	txn.MinerFees = []types.Currency{txFee}

	// Make sure to set the data
	txn.ArbitraryData = data

	// sign transaction
	if err := w.signTxn(txn, outputs); err != nil {
		return types.TransactionID{}, false, err
	}

	// finally commit
	txid, err := w.backend.SendTxn(txn)
	return txid, fundsLeft, err
}

// signTxn signs a transaction
func (w *Wallet) signTxn(txn types.Transaction, usedOutputIDs SpendableOutputs) error {
	// sign every coin input
	for idx, input := range txn.CoinInputs {
		// coinOutput has been checked during creation time, in the parent function,
		// hence we no longer need to check it here
		key := w.keys[usedOutputIDs[input.ParentID].Condition.UnlockHash()]
		err := input.Fulfillment.Sign(types.FulfillmentSignContext{
			ExtraObjects: []interface{}{uint64(idx)},
			Transaction:  txn,
			Key:          key.SecretKey,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
