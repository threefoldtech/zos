package types

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	rivtypes "github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/pkg/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Setup sets up indexes for types, must be called at least
// Onetime during the life time of the object
func Setup(ctx context.Context, db *mongo.Database) error {
	escrow := db.Collection(EscrowCollection)
	fmt.Println(escrow)
	_, err := escrow.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"_id": 1},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize reservation payment index")
	}

	return err
}

func AddTestReservation(ctx context.Context, db *mongo.Database) error {
	uh := rivtypes.UnlockHash{}
	err := uh.LoadString("01b20e5cfd38dd4f3fe85d04b43a04d603e8edb2c3978d6645c9fa94399c7e9603e5bf076b90e0")
	if err != nil {
		return err
	}
	info := ReservationPaymentInformation{
		reservationID: 1,
		expiration:    schema.Date{time.Now().Add(time.Hour * 6)},
		paid:          false,
		infos: []info{{
			farmerID:      schema.ID(5),
			escrowAddress: uh,
			totalAmount:   rivtypes.NewCurrency64(500004),
		}},
	}
	err = ReservationPaymentInfoCreate(ctx, db, info)
	if err != nil {
		return err
	}
	return nil
}
