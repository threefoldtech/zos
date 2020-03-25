package types

import (
	"context"
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
	err := uh.LoadString("018a65b9dc7b3e769a3fee8c06d04bbee6d77c94b29bd735e4eb5d81813886bb885fd9c9fa23e4")
	if err != nil {
		return err
	}
	info := ReservationPaymentInformation{
		ReservationID: 1,
		Expiration:    schema.Date{time.Now().Add(time.Hour * 6)},
		Paid:          false,
		Infos: []info{{
			FarmerID:      schema.ID(5),
			EscrowAddress: Address{uh},
			TotalAmount:   Currency{rivtypes.NewCurrency64(500004)},
		}},
	}
	err = ReservationPaymentInfoCreate(ctx, db, info)
	if err != nil {
		return err
	}
	return nil
}
