package stellar

import (
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	horizoneffects "github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/threefoldtech/zos/pkg/schema"
)

const (
	assetCode          = "TFT"
	assetIssuerTestnet = "GA47YZA3PKFUZMPLQ3B5F2E3CJIB57TGGU7SPCQT2WAEYKN766PWIMB3"
	assetIssuerProd    = "GBOVQKJYHXRR3DX6NOX2RRYFRCUMSADGDESTDNBDS6CDVLGVESRTAC47"
)

// ErrInsuficientBalance is an error that is used when there is insufficient balance
var ErrInsuficientBalance = errors.New("insuficient balance")

// Wallet is the foundation wallet
// Payments will be funded and fees will be taken with this wallet
type Wallet struct {
	keypair *keypair.Full
	network string
}

// New from seed
func New(seed string, network string) (*Wallet, error) {
	kp, err := keypair.ParseFull(seed)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		keypair: kp,
		network: network,
	}, nil
}

// CreateAccount and activates
func (w *Wallet) CreateAccount() (keypair.Full, error) {
	client, err := w.getHorizonClient()
	if err != nil {
		return keypair.Full{}, err
	}
	newKp, err := keypair.Random()
	if err != nil {
		return keypair.Full{}, err
	}

	sourceAccount, err := w.getAccountDetails(w.keypair.Address())
	if err != nil {
		return keypair.Full{}, errors.Wrap(err, "failed to get source account")
	}
	createAccountOp := txnbuild.CreateAccount{
		Destination: newKp.Address(),
		Amount:      "10",
	}
	tx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&createAccountOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       w.getNetworkPassPhrase(),
	}

	txeBase64, err := tx.BuildSignEncode(w.keypair)

	// Submit the transaction
	_, err = client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		return keypair.Full{}, errors.Wrap(hError, "error submitting transaction")
	}

	// Set the trustline
	sourceAccount, err = w.getAccountDetails(newKp.Address())
	changeTrustOp := txnbuild.ChangeTrust{
		SourceAccount: &sourceAccount,
		Line: txnbuild.CreditAsset{
			Code:   assetCode,
			Issuer: w.getIssuer(),
		},
		Limit: "10000",
	}
	trustTx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&changeTrustOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       w.getNetworkPassPhrase(),
	}

	txeBase64, err = trustTx.BuildSignEncode(newKp)
	// Submit the transaction
	_, err = client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		return keypair.Full{}, errors.Wrap(hError, "error submitting transaction")
	}

	return *newKp, nil
}

// GetBalance gets balance
func (w *Wallet) GetBalance(address string, id schema.ID) (xdr.Int64, error) {
	var total xdr.Int64
	horizonClient, err := w.getHorizonClient()
	if err != nil {
		return 0, err
	}

	txReq := horizonclient.TransactionRequest{
		ForAccount: address,
	}
	txes, err := horizonClient.Transactions(txReq)
	for _, tx := range txes.Embedded.Records {
		if tx.Memo == strconv.FormatInt(int64(id), 10) {
			effectsReq := horizonclient.EffectRequest{
				ForTransaction: tx.Hash,
			}
			effects, err := horizonClient.Effects(effectsReq)
			if err != nil {
				log.Debug().Msgf("failed to get transaction effects: %v", err)
				continue
			}
			for _, effect := range effects.Embedded.Records {
				if effect.GetAccount() != address {
					continue
				}
				if effect.GetType() == "account_credited" {
					// TODO also parse debits and payment to farmer
					creditedEffect := effect.(horizoneffects.AccountCredited)
					parsedAmount, err := amount.Parse(creditedEffect.Amount)
					if err != nil {
						continue
					}
					total += parsedAmount
				}
			}
		}
	}
	return total, nil
}

// Refund using a keypair
// keypair is account assiociated with farmer - user
// destination is the refund destination address
// id is the reservation ID to refund for
// TODO add a method to fund payment and refunds
func (w *Wallet) Refund(keypair keypair.Full, destination string, id schema.ID) (txnbuild.Transaction, error) {
	sourceAccount, err := w.getAccountDetails(keypair.Address())
	if err != nil {
		return txnbuild.Transaction{}, errors.Wrap(err, "failed to get source account")
	}
	amount, err := w.GetBalance(keypair.Address(), id)
	if err != nil {
		return txnbuild.Transaction{}, errors.Wrap(err, "failed to get balance")
	}
	// if no balance for this reservation, do nothing
	if amount == 0 {
		return txnbuild.Transaction{}, nil
	}

	paymentOP := txnbuild.Payment{
		Destination: destination,
		Amount:      strconv.FormatInt(int64(amount), 10),
		Asset: txnbuild.CreditAsset{
			Code:   assetCode,
			Issuer: w.getIssuer(),
		},
		SourceAccount: &sourceAccount,
	}

	formattedMemo := fmt.Sprintf("refund %d", id)
	memo := txnbuild.MemoText(formattedMemo)
	return txnbuild.Transaction{
		Operations: []txnbuild.Operation{&paymentOP},
		Timebounds: txnbuild.NewTimeout(300),
		Network:    w.getNetworkPassPhrase(),
		Memo:       memo,
	}, nil
}

// PayoutFarmer using a keypair
// keypair is account assiociated with farmer - user
// destination is the farmer destination address
// id is the reservation ID to pay for
func (w *Wallet) PayoutFarmer(keypair keypair.Full, destination string, amount xdr.Int64, id schema.ID) (txnbuild.Transaction, error) {
	sourceAccount, err := w.getAccountDetails(keypair.Address())
	if err != nil {
		return txnbuild.Transaction{}, errors.Wrap(err, "failed to get source account")
	}
	balance, err := w.GetBalance(keypair.Address(), id)
	if err != nil {
		return txnbuild.Transaction{}, errors.Wrap(err, "failed to get balance")
	}
	if balance < amount {
		return txnbuild.Transaction{}, ErrInsuficientBalance
	}

	// 10% cut for the foundation
	/*
		Based on the way we calculate the cost of reservation we know it has at most
		6 digit precision whereas stellar has 7 digits precision.
		This means that any valid reservation must necessarily have a "0" as least
		significant digit (when expressed as `stropes` as is the case here).
		With this knowledge it is safe to perform the 90% cut as regular integer operations
		instead of using floating points which might lead to floating point errors
	*/
	if amount%10 != 0 {
		return txnbuild.Transaction{}, errors.New("invalid reservation cost")
	}
	foundationCut := amount / 10 * 1
	amountDue := amount / 10 * 9

	farmerPaymentOP := txnbuild.Payment{
		Destination: destination,
		Amount:      strconv.FormatInt(int64(amountDue), 10),
		Asset: txnbuild.CreditAsset{
			Code:   assetCode,
			Issuer: w.getIssuer(),
		},
		SourceAccount: &sourceAccount,
	}
	foundationPaymentOP := txnbuild.Payment{
		Destination: w.keypair.Address(),
		Amount:      strconv.FormatInt(int64(foundationCut), 10),
		Asset: txnbuild.CreditAsset{
			Code:   assetCode,
			Issuer: w.getIssuer(),
		},
		SourceAccount: &sourceAccount,
	}

	formattedMemo := fmt.Sprintf("refund %d", id)
	memo := txnbuild.MemoText(formattedMemo)
	return txnbuild.Transaction{
		Operations: []txnbuild.Operation{&farmerPaymentOP, &foundationPaymentOP},
		Timebounds: txnbuild.NewTimeout(300),
		Network:    w.getNetworkPassPhrase(),
		Memo:       memo,
	}, nil
}

// FundTransaction funds a transaction with the foundation wallet
// For every operation in the transaction, the fee will be paid by the foundation wallet
func (w *Wallet) FundTransaction(tx *txnbuild.Transaction) (*txnbuild.Transaction, error) {
	sourceAccount, err := w.getAccountDetails(w.keypair.Address())
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

// SignTransaction sings of on a transaction with a given keypair
// and submits it to the network
func (w *Wallet) SignTransaction(keypair *keypair.Full, tx *txnbuild.Transaction) error {
	client, err := w.getHorizonClient()
	if err != nil {
		return errors.Wrap(err, "failed to get horizon client")
	}

	err = tx.Sign(keypair)
	if err != nil {
		return errors.Wrap(err, "failed to sign transaction with keypair")
	}

	// Submit the transaction
	_, err = client.SubmitTransaction(*tx)
	if err != nil {
		hError := err.(*horizonclient.Error)
		return errors.Wrap(hError, "error submitting transaction")
	}
	return nil
}

func (w *Wallet) getAccountDetails(address string) (account horizon.Account, err error) {
	client, err := w.getHorizonClient()
	if err != nil {
		return horizon.Account{}, err
	}
	ar := horizonclient.AccountRequest{AccountID: address}
	account, err = client.AccountDetail(ar)
	return
}

func (w *Wallet) getHorizonClient() (*horizonclient.Client, error) {
	switch w.network {
	case "testnet":
		return horizonclient.DefaultTestNetClient, nil
	case "production":
		return horizonclient.DefaultPublicNetClient, nil
	default:
		return nil, errors.New("network is not supported")
	}
}

func (w *Wallet) getIssuer() string {
	switch w.network {
	case "testnet":
		return assetIssuerTestnet
	case "production":
		return assetIssuerProd
	default:
		return assetIssuerTestnet
	}
}

func (w *Wallet) getNetworkPassPhrase() string {
	switch w.network {
	case "testnet":
		return network.TestNetworkPassphrase
	case "production":
		return network.PublicNetworkPassphrase
	default:
		return network.TestNetworkPassphrase
	}
}
