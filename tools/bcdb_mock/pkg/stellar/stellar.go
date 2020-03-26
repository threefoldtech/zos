package stellar

import (
	"log"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/txnbuild"
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
	client := horizonclient.DefaultTestNetClient
	newKp, err := keypair.Random()
	if err != nil {
		return "", err
	}

	sourceAccount, err := getAccountDetails(w.keypair.Address())
	if err != nil {
		log.Fatal(err)
	}
	createAccountOp := txnbuild.CreateAccount{
		Destination: newKp.Address(),
		Amount:      "10",
	}
	tx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&createAccountOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txeBase64, err := tx.BuildSignEncode(w.keypair)
	log.Println("Transaction base64: ", txeBase64)

	// Submit the transaction
	_, err = client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		return "", errors.Wrap(hError, "error submitting transaction")
	}

	// Set the trustline
	sourceAccount, err = getAccountDetails(newKp.Address())
	changeTrustOp := txnbuild.ChangeTrust{
		SourceAccount: &sourceAccount,
		Line: txnbuild.CreditAsset{
			Code:   assetCode,
			Issuer: assetIssuerTestnet,
		},
		Limit: "10000",
	}
	trustTx := txnbuild.Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []txnbuild.Operation{&changeTrustOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
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

func getAccountDetails(address string) (account horizon.Account, err error) {
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: address}
	account, err = client.AccountDetail(ar)
	return
}
