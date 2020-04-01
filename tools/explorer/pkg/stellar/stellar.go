package stellar

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	hProtocol "github.com/stellar/go/protocols/horizon"
	horizoneffects "github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/threefoldtech/zos/pkg/schema"
)

const (
	// TFTCode is the asset code for TFT on stellar
	TFTCode          = "TFT"
	tftIssuerTestnet = "GA47YZA3PKFUZMPLQ3B5F2E3CJIB57TGGU7SPCQT2WAEYKN766PWIMB3"
	tftIssuerProd    = "GBOVQKJYHXRR3DX6NOX2RRYFRCUMSADGDESTDNBDS6CDVLGVESRTAC47"

	// FreeTFTCode is the asset code for TFT on stellar
	FreeTFTCode          = "FreeTFT"
	freeTftIssuerTestnet = "GBLDUINEFYTF7XEE7YNWA3JQS4K2VD37YU7I2YAE7R5AHZDKQXSS2J6R"
	freeTftIssuerProd    = "GCBGS5TFE2BPPUVY55ZPEMWWGR6CLQ7T6P46SOFGHXEBJ34MSP6HVEUT"

	stellarPrecision       = 1e7
	stellarPrecisionDigits = 7

	// NetworkProduction uses stellar production network
	NetworkProduction = "production"
	// NetworkTest uses stellar test network
	NetworkTest = "testnet"
	// NetworkDebug doesn't do validation, and always address validation is skipped
	// Only supported by the AddressValidator
	NetworkDebug = "debug"
)

type (
	assetCodeEnum string

	// Signers is a flag type for setting the signers on the escrow accounts
	Signers []string

	// Wallet is the foundation wallet
	// Payments will be funded and fees will be taken with this wallet
	Wallet struct {
		keypair *keypair.Full
		network string
		asset   assetCodeEnum
		signers Signers
	}
)

const (
	// TFT assetcode
	TFT assetCodeEnum = TFTCode
	// FreeTFT assetcode
	FreeTFT assetCodeEnum = FreeTFTCode
)

// ErrInsuficientBalance is an error that is used when there is insufficient balance
var ErrInsuficientBalance = errors.New("insuficient balance")

// PayoutInfo holds information about which address needs to receive how many funds
// for payment commands which take multiple receivers
type PayoutInfo struct {
	Address string
	Amount  xdr.Int64
}

// New from seed
func New(seed, network, asset string, signers Signers) (*Wallet, error) {
	kp, err := keypair.ParseFull(seed)
	if err != nil {
		return nil, err
	}

	if len(signers) < 3 {
		log.Warn().Msg("to enable escrow account recovery, provide atleast 3 signers")
	}

	return &Wallet{
		keypair: kp,
		network: network,
		asset:   assetCodeEnum(asset),
		signers: signers,
	}, nil
}

// CreateAccount creates a new keypair
// and activates this keypair
// sets up a trustline to the correct issuer and asset
// and sets up multisig on the account for recovery of funds
func (w *Wallet) CreateAccount() (keypair.Full, error) {
	client, err := w.GetHorizonClient()
	if err != nil {
		return keypair.Full{}, err
	}
	newKp, err := keypair.Random()
	if err != nil {
		return keypair.Full{}, err
	}

	sourceAccount, err := w.GetAccountDetails(w.keypair.Address())
	if err != nil {
		return keypair.Full{}, errors.Wrap(err, "failed to get source account")
	}

	err = w.activateEscrowAccount(newKp, sourceAccount, client)
	if err != nil {
		return keypair.Full{}, errors.Wrapf(err, "failed to activate escrow account %s", newKp.Address())
	}

	err = w.setupTrustline(newKp, sourceAccount, client)
	if err != nil {
		return keypair.Full{}, errors.Wrapf(err, "failed to get setup trustline on escrow account %s", newKp.Address())
	}

	err = w.setupEscrowMultisig(newKp, sourceAccount, client)
	if err != nil {
		return keypair.Full{}, errors.Wrapf(err, "failed to get setup multsig on escrow account %s", newKp.Address())
	}

	return *newKp, nil
}

func (w *Wallet) activateEscrowAccount(newKp *keypair.Full, sourceAccount hProtocol.Account, client *horizonclient.Client) error {
	createAccountOp := txnbuild.CreateAccount{
		Destination: newKp.Address(),
		Amount:      "10",
	}
	tx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&createAccountOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       w.GetNetworkPassPhrase(),
	}

	txeBase64, err := tx.BuildSignEncode(w.keypair)
	if err != nil {
		return errors.Wrap(err, "failed to get build transaction")
	}

	// Submit the transaction
	_, err = client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		return errors.Wrap(hError, "error submitting transaction")
	}
	return nil
}

func (w *Wallet) setupTrustline(newKp *keypair.Full, sourceAccount hProtocol.Account, client *horizonclient.Client) error {
	sourceAccount, err := w.GetAccountDetails(newKp.Address())
	if err != nil {
		return errors.Wrap(err, "failed to get account details")
	}
	changeTrustOp := txnbuild.ChangeTrust{
		SourceAccount: &sourceAccount,
		Line: txnbuild.CreditAsset{
			Code:   w.asset.String(),
			Issuer: w.GetIssuer(),
		},
	}
	trustTx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&changeTrustOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       w.GetNetworkPassPhrase(),
	}

	txeBase64, err := trustTx.BuildSignEncode(newKp)
	if err != nil {
		return errors.Wrap(err, "failed to get build transaction")
	}

	// Submit the transaction
	_, err = client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		return errors.Wrap(hError.Problem, "error submitting transaction")
	}
	return nil
}

func (w *Wallet) setupEscrowMultisig(newKp *keypair.Full, sourceAccount hProtocol.Account, client *horizonclient.Client) error {
	if len(w.signers) >= 3 {
		// set the threshold for the master key equal to the amount of signers
		threshold := txnbuild.Threshold(len(w.signers))

		// set the threshold to complete transaction for signers. atleast 3 signatures are required
		txThreshold := txnbuild.Threshold(3)
		if len(w.signers) > 3 {
			txThreshold = txnbuild.Threshold(len(w.signers)/2 + 1)
		}

		var operations []txnbuild.Operation
		// add the signing options
		addSignersOp := txnbuild.SetOptions{
			LowThreshold:    txnbuild.NewThreshold(0),
			MediumThreshold: txnbuild.NewThreshold(txThreshold),
			HighThreshold:   txnbuild.NewThreshold(txThreshold),
			MasterWeight:    txnbuild.NewThreshold(threshold),
		}
		operations = append(operations, &addSignersOp)

		// add the signers
		for _, signer := range w.signers {
			addSignerOperation := txnbuild.SetOptions{
				Signer: &txnbuild.Signer{
					Address: signer,
					Weight:  1,
				},
			}
			operations = append(operations, &addSignerOperation)
		}

		addSignersTx := txnbuild.Transaction{
			SourceAccount: &sourceAccount,
			Operations:    operations,
			Timebounds:    txnbuild.NewTimeout(300),
			Network:       w.GetNetworkPassPhrase(),
		}

		txeBase64, err := addSignersTx.BuildSignEncode(newKp)
		if err != nil {
			return errors.Wrap(err, "failed to get build transaction")
		}

		// Submit the transaction
		_, err = client.SubmitTransactionXDR(txeBase64)
		if err != nil {
			hError := err.(*horizonclient.Error)
			return errors.Wrap(hError.Problem, "error submitting transaction")
		}
	}

	return nil
}

// KeyPairFromSeed parses a seed and creates a keypair for it, which can be
// used to sign transactions
func (w *Wallet) KeyPairFromSeed(seed string) (*keypair.Full, error) {
	return keypair.ParseFull(seed)
}

// GetBalance gets balance for an address and a given reservation id. It also returns
// a list of addresses which funded the given address.
func (w *Wallet) GetBalance(address string, id schema.ID) (xdr.Int64, []string, error) {

	if address == "" {
		err := fmt.Errorf("trying to get the balance of an empty address. this should never happen")
		log.Warn().Err(err).Send()
		return 0, nil, err
	}

	var total xdr.Int64
	horizonClient, err := w.GetHorizonClient()
	if err != nil {
		return 0, nil, err
	}

	cursor := ""

	txReq := horizonclient.TransactionRequest{
		ForAccount: address,
		Cursor:     cursor,
	}

	log.Info().Str("address", address).Msg("fetching balance for address")
	txes, err := horizonClient.Transactions(txReq)
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not get transactions")
	}

	donors := make(map[string]struct{})
	for len(txes.Embedded.Records) != 0 {
		for _, tx := range txes.Embedded.Records {
			if tx.Memo == strconv.FormatInt(int64(id), 10) {
				effectsReq := horizonclient.EffectRequest{
					ForTransaction: tx.Hash,
				}
				effects, err := horizonClient.Effects(effectsReq)
				if err != nil {
					log.Error().Err(err).Msgf("failed to get transaction effects")
					continue
				}
				// first check if we have been paid
				var isFunding bool
				for _, effect := range effects.Embedded.Records {
					if effect.GetAccount() != address {
						continue
					}
					if effect.GetType() == "account_credited" {
						isFunding = true
						creditedEffect := effect.(horizoneffects.AccountCredited)
						parsedAmount, err := amount.Parse(creditedEffect.Amount)
						if err != nil {
							continue
						}
						total += parsedAmount
					} else if effect.GetType() == "account_debited" {
						isFunding = false
						debitedEffect := effect.(horizoneffects.AccountDebited)
						parsedAmount, err := amount.Parse(debitedEffect.Amount)
						if err != nil {
							continue
						}
						total -= parsedAmount
					}
				}
				if isFunding {
					for _, effect := range effects.Embedded.Records {
						if effect.GetType() == "account_debited" && effect.GetAccount() != address {
							donors[effect.GetAccount()] = struct{}{}
						}
					}
				}
			}
			cursor = tx.PagingToken()
		}
		txReq.Cursor = cursor
		log.Info().Str("address", address).Msgf("fetching balance for address with cursor: %s", cursor)
		txes, err = horizonClient.Transactions(txReq)
		if err != nil {
			return 0, nil, errors.Wrap(err, "could not get transactions")
		}
	}

	donorList := []string{}
	for donor := range donors {
		donorList = append(donorList, donor)
	}
	log.Info().
		Int64("balance", int64(total)).
		Str("address", address).
		Int64("id", int64(id)).Msgf("status of balance for reservation")
	return total, donorList, nil
}

// Refund using a keypair
// keypair is account associated with farmer - user
// refund destination is the first address in the "funder" list as returned by
// GetBalance
// id is the reservation ID to refund for
func (w *Wallet) Refund(keypair keypair.Full, id schema.ID) error {
	sourceAccount, err := w.GetAccountDetails(keypair.Address())
	if err != nil {
		return errors.Wrap(err, "failed to get source account")
	}
	amount, funders, err := w.GetBalance(keypair.Address(), id)
	if err != nil {
		return errors.Wrap(err, "failed to get balance")
	}
	// if no balance for this reservation, do nothing
	if amount == 0 {
		return nil
	}
	destination := funders[0]

	paymentOP := txnbuild.Payment{
		Destination: destination,
		Amount:      big.NewRat(int64(amount), stellarPrecision).FloatString(stellarPrecisionDigits),
		Asset: txnbuild.CreditAsset{
			Code:   w.asset.String(),
			Issuer: w.GetIssuer(),
		},
		SourceAccount: &sourceAccount,
	}

	formattedMemo := fmt.Sprintf("%d", id)
	memo := txnbuild.MemoText(formattedMemo)
	tx := txnbuild.Transaction{
		Operations: []txnbuild.Operation{&paymentOP},
		Timebounds: txnbuild.NewTimeout(300),
		Network:    w.GetNetworkPassPhrase(),
		Memo:       memo,
	}

	fundedTx, err := w.fundTransaction(&tx)
	if err != nil {
		return errors.Wrap(err, "failed to fund transaction")
	}

	err = w.signAndSubmitTx(&keypair, fundedTx)
	if err != nil {
		return errors.Wrap(err, "failed to sign and submit transaction")
	}
	return nil
}

// PayoutFarmers using a keypair
// keypair is account assiociated with farmer - user
// destination is the farmer destination address
// id is the reservation ID to pay for
func (w *Wallet) PayoutFarmers(keypair keypair.Full, destinations []PayoutInfo, id schema.ID) error {
	sourceAccount, err := w.GetAccountDetails(keypair.Address())
	if err != nil {
		return errors.Wrap(err, "failed to get source account")
	}
	balance, _, err := w.GetBalance(keypair.Address(), id)
	if err != nil {
		return errors.Wrap(err, "failed to get balance")
	}
	requiredAmount := xdr.Int64(0)
	for _, pi := range destinations {
		requiredAmount += pi.Amount
	}
	if balance < requiredAmount {
		return ErrInsuficientBalance
	}

	paymentOps := make([]txnbuild.Operation, 0, len(destinations)+1)
	foundationCut := xdr.Int64(0)

	for _, pi := range destinations {
		// 10% cut for the foundation
		/*
			Based on the way we calculate the cost of reservation we know it has at most
			6 digit precision whereas stellar has 7 digits precision.
			This means that any valid reservation must necessarily have a "0" as least
			significant digit (when expressed as `stropes` as is the case here).
			With this knowledge it is safe to perform the 90% cut as regular integer operations
			instead of using floating points which might lead to floating point errors
		*/
		if pi.Amount%10 != 0 {
			return errors.New("invalid reservation cost")
		}
		foundationCut += pi.Amount / 10 * 1
		amountDue := pi.Amount / 10 * 9

		paymentOps = append(paymentOps, &txnbuild.Payment{
			Destination: pi.Address,
			Amount:      big.NewRat(int64(amountDue), stellarPrecision).FloatString(stellarPrecisionDigits),
			Asset: txnbuild.CreditAsset{
				Code:   w.asset.String(),
				Issuer: w.GetIssuer(),
			},
			SourceAccount: &sourceAccount,
		})
	}

	// add foundation payment
	paymentOps = append(paymentOps, &txnbuild.Payment{
		Destination: w.keypair.Address(),
		Amount:      big.NewRat(int64(foundationCut), stellarPrecision).FloatString(stellarPrecisionDigits),
		Asset: txnbuild.CreditAsset{
			Code:   w.asset.String(),
			Issuer: w.GetIssuer(),
		},
		SourceAccount: &sourceAccount,
	})

	formattedMemo := fmt.Sprintf("%d", id)
	memo := txnbuild.MemoText(formattedMemo)
	tx := txnbuild.Transaction{
		Operations: paymentOps,
		Timebounds: txnbuild.NewTimeout(300),
		Network:    w.GetNetworkPassPhrase(),
		Memo:       memo,
	}

	fundedTx, err := w.fundTransaction(&tx)
	if err != nil {
		return errors.Wrap(err, "failed to fund transaction")
	}

	err = w.signAndSubmitTx(&keypair, fundedTx)
	if err != nil {
		return errors.Wrap(err, "failed to sign and submit transaction")
	}
	return nil
}

// fundTransaction funds a transaction with the foundation wallet
// For every operation in the transaction, the fee will be paid by the foundation wallet
func (w *Wallet) fundTransaction(tx *txnbuild.Transaction) (*txnbuild.Transaction, error) {
	sourceAccount, err := w.GetAccountDetails(w.keypair.Address())
	if err != nil {
		return &txnbuild.Transaction{}, errors.Wrap(err, "failed to get source account")
	}

	// set the source account of the tx to the foundation account
	tx.SourceAccount = &sourceAccount

	if len(tx.Operations) == 0 {
		return &txnbuild.Transaction{}, errors.New("no operations were set on the transaction")
	}

	// calculate total fee based on the operations in the transaction
	tx.BaseFee = tx.BaseFee * uint32(len(tx.Operations))
	err = tx.Build()
	if err != nil {
		return &txnbuild.Transaction{}, errors.Wrap(err, "failed to build transaction")
	}

	err = tx.Sign(w.keypair)
	if err != nil {
		return &txnbuild.Transaction{}, errors.Wrap(err, "failed to sign transaction")
	}

	return tx, nil
}

// signAndSubmitTx sings of on a transaction with a given keypair
// and submits it to the network
func (w *Wallet) signAndSubmitTx(keypair *keypair.Full, tx *txnbuild.Transaction) error {
	client, err := w.GetHorizonClient()
	if err != nil {
		return errors.Wrap(err, "failed to get horizon client")
	}

	err = tx.Sign(keypair)
	if err != nil {
		return errors.Wrap(err, "failed to sign transaction with keypair")
	}

	log.Info().Msg("submitting transaction to the stellar network")
	// Submit the transaction
	_, err = client.SubmitTransaction(*tx)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Debug().
			Err(fmt.Errorf("%+v", hError.Problem.Extras)).
			Msg("error submitting transaction")
		return errors.Wrap(hError.Problem, "error submitting transaction")
	}
	return nil
}

// GetAccountDetails gets account details based an a Stellar address
func (w *Wallet) GetAccountDetails(address string) (account hProtocol.Account, err error) {
	client, err := w.GetHorizonClient()
	if err != nil {
		return hProtocol.Account{}, err
	}
	ar := horizonclient.AccountRequest{AccountID: address}
	log.Info().Str("address", address).Msgf("fetching account details for address: ")
	account, err = client.AccountDetail(ar)
	if err != nil {
		return hProtocol.Account{}, errors.Wrapf(err, "failed to get account details for account: %s", address)
	}
	return account, nil
}

// GetHorizonClient gets the horizon client based on the wallet's network
func (w *Wallet) GetHorizonClient() (*horizonclient.Client, error) {
	switch w.network {
	case "testnet":
		return horizonclient.DefaultTestNetClient, nil
	case "production":
		return horizonclient.DefaultPublicNetClient, nil
	default:
		return nil, errors.New("network is not supported")
	}
}

// GetIssuer gets the issuer based on the wallet's asset and network
func (w *Wallet) GetIssuer() string {
	switch w.asset {
	case TFT:
		switch w.network {
		case "testnet":
			return tftIssuerTestnet
		case "production":
			return tftIssuerProd
		default:
			return tftIssuerTestnet
		}
	case FreeTFT:
		switch w.network {
		case "testnet":
			return freeTftIssuerTestnet
		case "production":
			return freeTftIssuerProd
		default:
			return freeTftIssuerTestnet
		}
	default:
		return tftIssuerTestnet
	}
}

// GetNetworkPassPhrase gets the Stellar network passphrase based on the wallet's network
func (w *Wallet) GetNetworkPassPhrase() string {
	switch w.network {
	case "testnet":
		return network.TestNetworkPassphrase
	case "production":
		return network.PublicNetworkPassphrase
	default:
		return network.TestNetworkPassphrase
	}
}

func (e assetCodeEnum) String() string {
	switch e {
	case TFT:
		return TFTCode
	case FreeTFT:
		return FreeTFTCode
	}
	return "UNKNOWN"
}

func (i *Signers) String() string {
	repr := ""
	for _, s := range *i {
		repr += fmt.Sprintf("%s ", s)
	}
	return repr
}

// Set a value on the signers flag
func (i *Signers) Set(value string) error {
	*i = append(*i, value)
	return nil
}
