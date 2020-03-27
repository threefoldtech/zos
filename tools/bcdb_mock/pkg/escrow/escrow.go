package escrow

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory"
	directorytypes "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/escrow/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/stellar"
	workloadtypes "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/workloads/types"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	// Escrow service manages a dedicate wallet for payments for reservations.
	Escrow struct {
		wallet *stellar.Wallet
		db     *mongo.Database

		reservationChannel chan reservationRegisterJob
		deployedChannel    chan schema.ID
		cancelledChannel   chan schema.ID

		nodeAPI NodeAPI
		farmAPI FarmAPI

		ctx context.Context
	}

	// NodeAPI operations on node database
	NodeAPI interface {
		// Get a node from the database using its ID
		Get(ctx context.Context, db *mongo.Database, id string, proofs bool) (directorytypes.Node, error)
	}

	// FarmAPI operations on farm database
	FarmAPI interface {
		// GetByID get a farm from the database using its ID
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

const (
	// interval between every check of active escrow accounts
	balanceCheckInterval = time.Minute * 1
)

// New creates a new escrow object and fetches all addresses for the escrow wallet
func New(wallet *stellar.Wallet, db *mongo.Database) *Escrow {
	jobChannel := make(chan reservationRegisterJob)
	deployChannel := make(chan schema.ID)
	cancelChannel := make(chan schema.ID)

	return &Escrow{
		wallet:             wallet,
		db:                 db,
		nodeAPI:            &directory.NodeAPI{},
		farmAPI:            &directory.FarmAPI{},
		reservationChannel: jobChannel,
		deployedChannel:    deployChannel,
		cancelledChannel:   cancelChannel,
	}
}

// Run the escrow until the context is done
func (e *Escrow) Run(ctx context.Context) error {
	ticker := time.NewTicker(balanceCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C: // check reservations, mark those which are funded
			log.Debug().Msg("scanning active ecrow accounts balance")
			// load active escrows
			reservationEscrows, err := types.GetAllActiveReservationPaymentInfos(ctx, e.db)
			if err != nil {
				log.Error().Msgf("failed to load active reservations from escrow: %s", err)
				continue
			}
			for _, escrowInfo := range reservationEscrows {
				allPaid := true
				for _, escrowAccount := range escrowInfo.Infos {
					balance, _, err := e.wallet.GetBalance(escrowAccount.EscrowAddress, escrowInfo.ReservationID)
					if err != nil {
						allPaid = false
						log.Error().Msgf("failed to verify escrow account balance: %s", err)
						break
					}
					if balance < escrowAccount.TotalAmount {
						allPaid = false
						log.Debug().Msgf("escrow account %s for reservation id %d is not funded yet", escrowAccount.EscrowAddress, escrowInfo.ReservationID)
						break
					}
				}
				if allPaid {
					// TODO: check reservation state, if "PAY" -> "DEPLOY"
					reservation, err := workloadtypes.ReservationGetByID(ctx, e.db, escrowInfo.ReservationID)
					if err != nil {
						log.Error().Msgf("failed to load reservation: %s", err)
						continue
					}
					pl, err := workloadtypes.NewPipeline(reservation)
					if err != nil {
						log.Error().Msgf("failed to process reservation in pipeline: %s", err)
						continue
					}

					reservation, _ = pl.Next()
					if !reservation.IsAny(workloadtypes.Pay) {
						// TODO
					}

					// update reservation
					if err = workloadtypes.ReservationSetNextAction(ctx, e.db, escrowInfo.ReservationID, workloadtypes.Pay); err != nil {
						log.Error().Msgf("failed to set reservation in deploy state: %s", err)
						continue
					}

					escrowInfo.Paid = true
					if err = types.ReservationPaymentInfoUpdate(ctx, e.db, escrowInfo); err != nil {
						log.Error().Msgf("failed to mark reservation escrow info as paid: %s", err)
						continue
					}
				}
			}
		case job := <-e.reservationChannel:
			log.Debug().Msg("Processing new reservation escrow")
			rsuPerFarmer, err := e.processReservation(job.reservation.DataReservation)
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
				var address string
				address, err = e.createOrLoadAccount(farmer, job.reservation.CustomerTid)
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
		case id := <-e.deployedChannel:
			rpi, err := types.ReservationPaymentInfoGet(ctx, e.db, id)
			if err != nil {
				log.Error().Msgf("failed to get reservation escrow info: %s", err)
				continue
			}
			// we already verified we have enough balance on every escrow for this reservation
			for _, escrowDetails := range rpi.Infos {
				// in case of an error in this flow we continue, so we try to pay as much
				// farmers as possible even if one fails
				farm, err := e.farmAPI.GetByID(ctx, e.db, int64(escrowDetails.FarmerID))
				if err != nil {
					log.Error().Msgf("failed to load farm info: %s", err)
					continue
				}
				// TODO rework this type to an object with currency and filter based on "TFT"
				destination := farm.WalletAddresses[0]
				addressInfo, err := types.GetByAddress(ctx, e.db, escrowDetails.EscrowAddress)
				if err != nil {
					log.Error().Msgf("failed to load escrow address info: %s", err)
					continue
				}
				kp, err := e.wallet.KeyPairFromSeed(addressInfo.Secret)
				if err != nil {
					log.Error().Msgf("failed to parse escrow address secret: %s", err)
					continue
				}
				if err = e.wallet.PayoutFarmer(*kp, destination, escrowDetails.TotalAmount, id); err != nil {
					log.Error().Msgf("failed to pay farmer: %s", err)
					continue
				}
				// now refund any possible overpayment
				if err = e.wallet.Refund(*kp, id); err != nil {
					log.Error().Msgf("failed to pay farmer: %s", err)
					continue
				}
			}
			rpi.Released = true
			if err = types.ReservationPaymentInfoUpdate(ctx, e.db, rpi); err != nil {
				log.Error().Msgf("could not mark escrows for %d as released: %s", rpi.ReservationID, err)
			}
		case id := <-e.cancelledChannel:
			rpi, err := types.ReservationPaymentInfoGet(ctx, e.db, id)
			if err != nil {
				log.Error().Msgf("failed to get reservation escrow info: %s", err)
				continue
			}
			for _, escrowDetails := range rpi.Infos {
				// in case of an error in this flow we continue, so we try to pay as much
				// farmers as possible even if one fails
				addressInfo, err := types.GetByAddress(ctx, e.db, escrowDetails.EscrowAddress)
				if err != nil {
					log.Error().Msgf("failed to load escrow address info: %s", err)
					continue
				}
				kp, err := e.wallet.KeyPairFromSeed(addressInfo.Secret)
				if err != nil {
					log.Error().Msgf("failed to parse escrow address secret: %s", err)
					continue
				}
				if err = e.wallet.Refund(*kp, id); err != nil {
					log.Error().Msgf("failed to pay farmer: %s", err)
					continue
				}
			}
			rpi.Canceled = true
			if err = types.ReservationPaymentInfoUpdate(ctx, e.db, rpi); err != nil {
				log.Error().Msgf("could not mark escrows for %d as canceled: %s", rpi.ReservationID, err)
			}
		}
	}
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

// ReservationDeployed informs the escrow that a reservation has been successfully
// deployed, so the escrow can release the funds to the farmer (and refund any excess)
func (e *Escrow) ReservationDeployed(reservationID schema.ID) {
	e.deployedChannel <- reservationID
}

// ReservationCanceled informs the escrow of a canceled reservation so it can refund
// the user
func (e *Escrow) ReservationCanceled(reservationID schema.ID) {
	e.cancelledChannel <- reservationID
}

// createOrLoadAccount creates or loads account based on farmer - customer id
func (e *Escrow) createOrLoadAccount(farmerID int64, customerTID int64) (string, error) {
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
