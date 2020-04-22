package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/pkg/errors"
	"github.com/stellar/go/xdr"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/tools/builders"

	"github.com/urfave/cli"
)

func cmdsProvision(c *cli.Context) error {
	var (
		seedPath   = mainSeed
		d          = c.String("duration")
		assets     = c.StringSlice("asset")
		volumes    = c.StringSlice("volume")
		containers = c.StringSlice("container")
		zdbs       = c.StringSlice("zdb")
		kubes      = c.StringSlice("kube")
		networks   = c.StringSlice("network")
		err        error
	)

	reservationBuilder := builders.NewReservationBuilder(bcdb, mainui)

	for _, vol := range volumes {
		f, err := os.Open(vol)
		if err != nil {
			return errors.Wrap(err, "failed to open volume")
		}

		volumeBuilder, err := builders.LoadVolumeBuilder(f)
		if err != nil {
			return errors.Wrap(err, "failed to load the reservation builder")
		}
		reservationBuilder.AddVolume(*volumeBuilder)
	}

	for _, cont := range containers {
		f, err := os.Open(cont)
		if err != nil {
			return errors.Wrap(err, "failed to open container")
		}

		containerBuilder, err := builders.LoadContainerBuilder(f)
		if err != nil {
			return errors.Wrap(err, "failed to load the reservation builder")
		}
		reservationBuilder.AddContainer(*containerBuilder)
	}

	for _, zdb := range zdbs {
		f, err := os.Open(zdb)
		if err != nil {
			return errors.Wrap(err, "failed to open zdb")
		}

		zdbBuilder, err := builders.LoadZdbBuilder(f)
		if err != nil {
			return errors.Wrap(err, "failed to load the zdb builder")
		}
		reservationBuilder.AddZdb(*zdbBuilder)
	}

	for _, k8s := range kubes {
		f, err := os.Open(k8s)
		if err != nil {
			return errors.Wrap(err, "failed to open kube")
		}

		k8sBuilder, err := builders.LoadK8sBuilder(f)
		if err != nil {
			return errors.Wrap(err, "failed to load the k8s builder")
		}
		reservationBuilder.AddK8s(*k8sBuilder)
	}

	for _, network := range networks {
		f, err := os.Open(network)
		if err != nil {
			return errors.Wrap(err, "failed to open reservation")
		}

		networkBuilder, err := builders.LoadNetworkBuilder(f)
		if err != nil {
			return errors.Wrap(err, "failed to load the network builder")
		}
		reservationBuilder.AddNetwork(*networkBuilder)
	}

	_, err = reservationBuilder.WithDuration(d)
	if err != nil {
		return errors.Wrap(err, "failed to set the reservation builder duration")
	}

	reservationBuilder.WithDryRun(true).WithSeedPath(seedPath).WithAssets(assets)

	response, err := reservationBuilder.Deploy()
	if err != nil {
		return errors.Wrap(err, "failed to deploy reservation")
	}

	totalAmount := xdr.Int64(0)
	for _, detail := range response.EscrowInformation.Details {
		totalAmount += detail.TotalAmount
	}

	fmt.Printf("Reservation for %v send to node bcdb\n", d)
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
	reservationBuilder := builders.NewReservationBuilder(bcdb, mainui)
	return reservationBuilder.DeleteReservation(c.Int64("reservation"))
}
