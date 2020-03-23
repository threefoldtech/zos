package escrow

import (
	rivtypes "github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/tfchain"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	Escrow struct {
		wallet tfchain.Wallet
		db     *mongo.Database
	}

	info struct {
		totalAmount   rivtypes.Currency
		escrowAddress rivtypes.UnlockHash
	}

	ReservationPaymentInformation struct {
		reservationID schema.ID
		infos         map[string]struct {
			info
		}
	}
)

func RegisterReservation(reservation *workloads.TfgridWorkloadsReservation1) error {
	return nil
}
