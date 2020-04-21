package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/xdr"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/builders"
	"github.com/threefoldtech/zos/tools/client"

	"github.com/urfave/cli"
)

var (
	day             = time.Hour * 24
	defaultDuration = day * 30
)

func encryptSecret(plain, nodeID string) (string, error) {
	if len(plain) == 0 {
		return "", nil
	}

	pubkey, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		return "", err
	}

	encrypted, err := crypto.Encrypt([]byte(plain), pubkey)
	return hex.EncodeToString(encrypted), err
}

func provisionCustomZDB(r *provision.Reservation) error {
	var config provision.ZDB
	if err := json.Unmarshal(r.Data, &config); err != nil {
		return errors.Wrap(err, "failed to load zdb reservation schema")
	}

	encrypted, err := encryptSecret(config.Password, r.NodeID)
	if err != nil {
		return err
	}

	config.Password = encrypted
	r.Data, err = json.Marshal(config)

	return err
}

func provisionCustomContainer(r *provision.Reservation) error {
	var config provision.Container
	var err error
	if err := json.Unmarshal(r.Data, &config); err != nil {
		return errors.Wrap(err, "failed to load zdb reservation schema")
	}

	if config.SecretEnv == nil {
		config.SecretEnv = make(map[string]string)
	}

	for k, v := range config.Env {
		v, err := encryptSecret(v, r.NodeID)
		if err != nil {
			return errors.Wrapf(err, "failed to encrypt env with key '%s'", k)
		}
		config.SecretEnv[k] = v
	}
	config.Env = make(map[string]string)
	r.Data, err = json.Marshal(config)

	return err
}

var (
	provCustomModifiers = map[provision.ReservationType]func(r *provision.Reservation) error{
		provision.ZDBReservation:       provisionCustomZDB,
		provision.ContainerReservation: provisionCustomContainer,
	}
)

func cmdsProvision(c *cli.Context) error {
	var (
		path     = c.String("schema")
		seedPath = mainSeed
		d        = c.String("duration")
		assets   = c.StringSlice("asset")
		duration time.Duration
		err      error
	)

	f, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "failed to open reservation")
	}

	volumeBuilder, err := builders.LoadVolumeBuilder(f)
	if err != nil {
		return errors.Wrap(err, "failed to load the reservation builder")
	}

	reservationBuilder, err := builders.NewReservationBuilder()
	if err != nil {
		return errors.Wrap(err, "failed to load the reservation builder")
	}

	reservationBuilder.AddVolume(*volumeBuilder)

	_, err = reservationBuilder.WithDuration(d)
	if err != nil {
		return errors.Wrap(err, "failed to set the reservation builder duration")
	}

	reservationBuilder.WithDryRun(true).WithSeedPath(seedPath).WithAssets(assets)

	response, err := reservationBuilder.Deploy(bcdb, mainui)
	if err != nil {
		return errors.Wrap(err, "failed to deploy reservation")
	}

	totalAmount := xdr.Int64(0)
	for _, detail := range response.EscrowInformation.Details {
		totalAmount += detail.TotalAmount
	}

	fmt.Printf("Reservation for %v send to node bcdb\n", duration)
	fmt.Printf("Resource: /reservations/%v\n", response.ID)
	fmt.Println()

	fmt.Printf("Reservation id: %d \n", response.ID)
	fmt.Printf("Asset to pay: %s\n", response.EscrowInformation.Asset)
	fmt.Printf("Reservation escrow address: %s \n", response.EscrowInformation.Address)
	fmt.Printf("Reservation amount: %s %s\n", formatCurrency(totalAmount), response.EscrowInformation.Asset.Code())

	for _, detail := range response.EscrowInformation.Details {
		fmt.Println()
		fmt.Printf("FarmerID: %v\n", detail.FarmerID)
		fmt.Printf("Amount: %s\n", formatCurrency(detail.TotalAmount))
	}

	return nil
}

func formatCurrency(amount xdr.Int64) string {
	currency := big.NewRat(int64(amount), 1e7)
	return currency.FloatString(7)
}

func embed(schema interface{}, t provision.ReservationType, node string) (*provision.Reservation, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	r := &provision.Reservation{
		NodeID: node,
		Type:   t,
		Data:   raw,
	}

	return r, nil
}

func cmdsDeleteReservation(c *cli.Context) error {
	var (
		resID  = c.Int64("reservation")
		userID = mainui.ThreebotID
		//seedPath = c.GlobalString("seed")
	)

	reservation, err := bcdb.Workloads.Get(schema.ID(resID))
	if err != nil {
		return errors.Wrap(err, "failed to get reservation info")
	}

	signer, err := client.NewSigner(mainui.Key().PrivateKey.Seed())
	if err != nil {
		return errors.Wrapf(err, "failed to load signer")
	}

	_, signature, err := signer.SignHex(resID, reservation.Json)
	if err != nil {
		return errors.Wrap(err, "failed to sign the reservation")
	}

	if err := bcdb.Workloads.SignDelete(schema.ID(resID), schema.ID(userID), signature); err != nil {
		return errors.Wrapf(err, "failed to sign deletion of reservation: %d", resID)
	}

	fmt.Printf("Reservation %v marked as to be deleted\n", resID)
	return nil
}
