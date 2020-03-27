package escrow

import (
	"context"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory"
	directorytypes "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/escrow/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/stellar"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	// Escrow service manages a dedicate wallet for payments for reservations.
	Escrow struct {
		wallet             *stellar.Wallet
		db                 *mongo.Database
		reservationChannel chan reservationRegisterJob
		// TODO: Remove
		farmAPI FarmAPI
	}

	// FarmAPI interface
	FarmAPI interface {
		GetByID(ctx context.Context, db *mongo.Database, id int64) (directorytypes.Farm, error)
	}

	reservationRegisterJob struct {
		reservation  workloads.Reservation
		responseChan chan reservationRegisterJobResponse
	}

	reservationRegisterJobResponse struct {
		data []types.EscrowDetail
		err  error
	}
)

// New creates a new escrow object and fetches all addresses for the escrow wallet
func New(wallet *stellar.Wallet, db *mongo.Database) (*Escrow, error) {
	jobChannel := make(chan reservationRegisterJob)
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
			rsuPerFarmer, err := e.processReservation(job.reservation.DataReservation, &dbNodeSource{ctx: ctx, db: e.db})
			if err != nil {
				job.responseChan <- reservationRegisterJobResponse{
					err: err,
				}
				close(job.responseChan)
				continue
			}
			res, err := e.calculateReservationCost(rsuPerFarmer)
			if err != nil {
				job.responseChan <- reservationRegisterJobResponse{
					err: err,
				}
				close(job.responseChan)
				continue
			}
			details := make([]types.EscrowDetail, 0, len(res))
			for farmer, value := range res {
				address, err := e.CreateOrLoadAccount(farmer, job.reservation.CustomerTid)
				if err != nil {
					job.responseChan <- reservationRegisterJobResponse{
						err: err,
					}
					close(job.responseChan)
					break
				}
				details = append(details, types.EscrowDetail{
					FarmerID:      schema.ID(farmer),
					EscrowAddress: address,
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

// CreateOrLoadAccount creates or loads account based on farmer - customer id
func (e *Escrow) CreateOrLoadAccount(farmerID int64, customerTID int64) (string, error) {
	res, err := types.Get(context.Background(), e.db, farmerID, customerTID)
	if err != nil {
		if err == types.ErrAddressNotFound {
			keypair, err := e.wallet.CreateAccount()
			if err != nil {
				return "", errors.Wrap(err, "failed to create a new account for farmer - customer")
			}
			err = types.FarmerCustomerAddressCreate(context.Background(), e.db, types.FarmerCustomerAddress{
				CustomerTID: customerTID,
				Address:     keypair.Address(),
				FarmerID:    farmerID,
				Secret:      keypair.Seed(),
			})
			if err != nil {
				return "", errors.Wrap(err, "failed to save a new account for farmer - customer")
			}
			return keypair.Address(), nil
		}
		return "", errors.Wrap(err, "failed to get farmer - customer address")
	}
	return res.Address, nil
}

// RegisterReservation registers a workload reservation
func (e *Escrow) RegisterReservation(reservation workloads.Reservation) ([]types.EscrowDetail, error) {
	job := reservationRegisterJob{
		reservation:  reservation,
		responseChan: make(chan reservationRegisterJobResponse),
	}
	e.reservationChannel <- job

	response := <-job.responseChan

	return response.data, response.err
}
