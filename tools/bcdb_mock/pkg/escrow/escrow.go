package escrow

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	rivclient "github.com/threefoldtech/rivine/pkg/client"
	rivtypes "github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/pkg/schema"
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
		responseChan chan reservationRegisterJobResponse
	}

	reservationRegisterJobResponse struct {
		data []types.EscrowDetail
		err  error
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
			rsuPerFarmer, err := processReservation(job.reservation.DataReservation, &dbNodeSource{ctx: ctx, db: e.db})
			if err != nil {
				job.responseChan <- reservationRegisterJobResponse{
					err: err,
				}
				close(job.responseChan)
				continue
			}
			res, err := e.CalculateReservationCost(rsuPerFarmer)
			if err != nil {
				job.responseChan <- reservationRegisterJobResponse{
					err: err,
				}
				close(job.responseChan)
				continue
			}
			details := make([]types.EscrowDetail, 0, len(res))
			for farmer, value := range res {
				var uh rivtypes.UnlockHash
				uh, err = e.wallet.GenerateAddress()
				if err != nil {
					job.responseChan <- reservationRegisterJobResponse{
						err: err,
					}
					close(job.responseChan)
					break
				}
				details = append(details, types.EscrowDetail{
					FarmerID:      schema.ID(farmer),
					EscrowAddress: types.Address{UnlockHash: uh},
					TotalAmount:   value,
				})
			}
			if err != nil {
				continue
			}
			reservationPaymentInfo := types.ReservationPaymentInformation{
				Infos:         details,
				ReservationID: job.reservation.ID,
				Expiration:    job.reservation.DataReservation.ExpirationProvisioning,
				Paid:          false,
			}
			err = types.ReservationPaymentInfoCreate(ctx, e.db, reservationPaymentInfo)
			job.responseChan <- reservationRegisterJobResponse{
				err:  err,
				data: details,
			}
		}
	}
}

// RegisterReservation registers a workload reservation
func (e *Escrow) RegisterReservation(reservation workloads.TfgridWorkloadsReservation1) ([]types.EscrowDetail, error) {
	job := reservationRegisterJob{
		reservation:  reservation,
		responseChan: make(chan reservationRegisterJobResponse),
	}
	e.reservationChannel <- job

	response := <-job.responseChan

	return response.data, response.err
}

// CalculateReservationCost calculates the cost of reservation based on a resource per farmer map
func (e *Escrow) CalculateReservationCost(rsuPerFarmerMap rsuPerFarmer) (map[int64]types.Currency, error) {
	costPerFarmerMap := make(map[int64]types.Currency)
	for id, rsu := range rsuPerFarmerMap {
		farm, err := e.farmAPI.GetByID(context.Background(), e.db, id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get farm with id: %d", id)
		}
		// why is this a list ?!
		if len(farm.ResourcePrices) == 0 {
			return nil, fmt.Errorf("farm with id: %d does not have price setup", id)
		}
		price := farm.ResourcePrices[0]
		cost := types.Currency{}

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

		cost = types.Currency{Currency: cost.Add(cruPriceCoin.Mul64(uint64(rsu.cru)))}
		cost = types.Currency{Currency: cost.Add(sruPriceCoin.Mul64(uint64(rsu.sru)))}
		cost = types.Currency{Currency: cost.Add(hruPriceCoin.Mul64(uint64(rsu.hru)))}
		cost = types.Currency{Currency: cost.Add(mruPriceCoin.Mul64(uint64(rsu.mru)))}

		costPerFarmerMap[id] = cost
	}
	return costPerFarmerMap, nil
}
