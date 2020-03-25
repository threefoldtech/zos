package escrow

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	rivclient "github.com/threefoldtech/rivine/pkg/client"
	rivtypes "github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory"
	directorytypes "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/escrow/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/tfchain"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	// Escrow service manages a dedicate wallet for payments for reservations.
	Escrow struct {
		wallet             tfchain.Wallet
		db                 *mongo.Database
		reservationChannel chan reservationRegisterJob
		farmAPI            FarmAPI
	}

	// FarmAPI interface
	FarmAPI interface {
		GetByID(ctx context.Context, db *mongo.Database, id int64) (directorytypes.Farm, error)
	}

	reservationRegisterJob struct {
		reservation  workloads.TfgridWorkloadsReservation1
		responseChan chan map[string]string
	}
)

// New creates a new escrow object and fetches all addresses for the escrow wallet
func New(wallet tfchain.Wallet, db *mongo.Database) (*Escrow, error) {
	jobChannel := make(chan reservationRegisterJob)
	addresses, err := types.GetAllAddresses(context.Background(), db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan for addresses")
	}
	// use inner type actually used by wallet
	addressesToScan := make([]rivtypes.UnlockHash, len(addresses))
	for i := range addresses {
		addressesToScan[i] = addresses[i].UnlockHash
	}
	err = wallet.LoadAddresses(addressesToScan)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load addresses")
	}
	return &Escrow{
		wallet:             wallet,
		db:                 db,
		farmAPI:            &directory.FarmAPI{},
		reservationChannel: jobChannel,
	}, nil
}

// Run the escrow until the context is done
func (e *Escrow) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case job := <-e.reservationChannel:
			processReservation(job.reservation.DataReservation)
		}
	}
}

// RegisterReservation registers a workload reservation
func RegisterReservation(reservation *workloads.TfgridWorkloadsReservation1) (map[string]string, error) {
	return nil, nil
}

// CalculateReservationCost calculates the cost of reservation based on a resource per farmer map
func (e *Escrow) CalculateReservationCost(rsuPerFarmerMap rsuPerFarmer) (map[int64]rivtypes.Currency, error) {
	costPerFarmerMap := make(map[int64]rivtypes.Currency)
	for id, rsu := range rsuPerFarmerMap {
		farm, err := e.farmAPI.GetByID(context.Background(), e.db, id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get farm with id: %d", id)
		}
		// why is this a list ?!
		if len(farm.ResourcePrices) == 0 {
			return nil, fmt.Errorf("Farm with id: %d does not have price setup", id)
		}
		price := farm.ResourcePrices[0]
		cost := rivtypes.Currency{}

		cc := rivclient.NewCurrencyConvertor(rivtypes.DefaultCurrencyUnits(), "TFT")
		cruPriceCoin, err := cc.ParseCoinString(strconv.FormatFloat(price.Cru, 'f', 9, 64))
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse cru price")
		}
		sruPriceCoin, err := cc.ParseCoinString(strconv.FormatFloat(price.Sru, 'f', 9, 64))
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse sru price")
		}
		hruPriceCoin, err := cc.ParseCoinString(strconv.FormatFloat(price.Hru, 'f', 9, 64))
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse hru price")
		}
		mruPriceCoin, err := cc.ParseCoinString(strconv.FormatFloat(price.Mru, 'f', 9, 64))
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse mru price")
		}

		cost = cost.Add(cruPriceCoin.Mul64(uint64(rsu.cru)))
		cost = cost.Add(sruPriceCoin.Mul64(uint64(rsu.sru)))
		cost = cost.Add(hruPriceCoin.Mul64(uint64(rsu.hru)))
		cost = cost.Add(mruPriceCoin.Mul64(uint64(rsu.mru)))

		costPerFarmerMap[id] = cost
	}
	return costPerFarmerMap, nil
}
