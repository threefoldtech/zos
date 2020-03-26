package stellar

import (
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

// Wallet is a stellar wallet
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
func (w *Wallet) CreateAccount() (string, error) {
	client, err := w.getHorizonClient()
	if err != nil {
		return "", err
	}
	newKp, err := keypair.Random()
	if err != nil {
		return "", err
	}

	sourceAccount, err := w.getAccountDetails(w.keypair.Address())
	if err != nil {
		return "", errors.Wrap(err, "failed to get source account")
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
		return "", errors.Wrap(hError, "error submitting transaction")
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
		return "", errors.Wrap(hError, "error submitting transaction")
	}

	return newKp.Address(), nil
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
