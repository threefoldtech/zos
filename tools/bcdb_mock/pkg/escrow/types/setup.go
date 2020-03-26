package types

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stellar/go/xdr"
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

// AddTestReservation helper method to inser a reservation for testing purposes
func AddTestReservation(ctx context.Context, db *mongo.Database) error {
	info := ReservationPaymentInformation{
		ReservationID: 1,
		Expiration:    schema.Date{Time: time.Now().Add(time.Hour * 6)},
		Paid:          false,
		Infos: []EscrowDetail{{
			FarmerID:      schema.ID(5),
			EscrowAddress: "GC27XOVPTZO4QB2VKKHQEMBDWHFSQT3JM4GH4LTQCIXAPK3IYXVCFOFI",
			TotalAmount:   xdr.Int64(500004),
		}},
	}
	err := ReservationPaymentInfoCreate(ctx, db, info)
	if err != nil {
		return err
	}
	return nil
}
