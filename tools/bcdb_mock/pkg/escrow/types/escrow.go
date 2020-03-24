package types

import (
	"context"
	"time"

	"github.com/pkg/errors"

	rivtypes "github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/pkg/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	// EscrowCollection db collection name
	EscrowCollection = "escrow"
)

var (
	// ErrEscrowExists is returned when trying to save escrow information for a
	// reservation that already has escrow information
	ErrEscrowExists = errors.New("escrow(s) for reservation already exists")
	// ErrEscrowNotFound is returned if escrow information is not found
	ErrEscrowNotFound = errors.New("escrow information not found")
)

type (
	ReservationPaymentInformation struct {
		ReservationID schema.ID   `bson:"_id"`
		Expiration    schema.Date `bson:"expiration"`
		Infos         []info      `bson:"infos"`
		Paid          bool        `bson:"paid"`
	}
	info struct {
		FarmerID      schema.ID           `bson:"farmer_id"`
		TotalAmount   rivtypes.Currency   `bson:"total_amount"`
		EscrowAddress rivtypes.UnlockHash `bson:"escrow_address"`
	}
)

// ReservationPaymentInfoCreate creates the reservation payment information
func ReservationPaymentInfoCreate(ctx context.Context, db *mongo.Database, reservationPaymentInfo ReservationPaymentInformation) error {
	col := db.Collection(EscrowCollection)
	_, err := col.InsertOne(ctx, reservationPaymentInfo)
	if err != nil {
		if merr, ok := err.(mongo.WriteException); ok {
			errCode := merr.WriteErrors[0].Code
			if errCode == 11000 {
				return ErrEscrowExists
			}
		}
		return err
	}
	return nil
}

// ReservationPaymentInfoUpdate update reservation payment info
func ReservationPaymentInfoUpdate(ctx context.Context, db *mongo.Database, update ReservationPaymentInformation) error {
	filter := bson.M{"_id": update.ReservationID}
	// actually update the user with final data
	if _, err := db.Collection(EscrowCollection).UpdateOne(ctx, filter, bson.M{"$set": update}); err != nil {
		return err
	}

	return nil
}

// GetAllActiveReservationPaymentInfos get all active reservation payment information
func GetAllActiveReservationPaymentInfos(ctx context.Context, db *mongo.Database) ([]ReservationPaymentInformation, error) {
	filter := bson.M{"paid": false, "expiration": bson.M{"$gt": schema.Date{time.Now()}}}
	cursor, err := db.Collection(EscrowCollection).Find(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get cursor over active payment infos")
	}
	paymentInfos := make([]ReservationPaymentInformation, 0)
	err = cursor.All(ctx, &paymentInfos)
	if err != nil {
		err = errors.Wrap(err, "Failed to decode active payment information")
	}
	return paymentInfos, err
}

func GetAllAddresses(ctx context.Context, db *mongo.Database) ([]rivtypes.UnlockHash, error) {
	cursor, err := db.Collection(EscrowCollection).Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create cursor over payment infos")
	}
	defer cursor.Close(ctx)
	unlockhashes := make([]rivtypes.UnlockHash, 0)
	var reservationPaymentInfo ReservationPaymentInformation
	for cursor.Next(ctx) {
		err = cursor.Decode(&reservationPaymentInfo)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to decode reservation payment info")
		}
		for _, paymentInfo := range reservationPaymentInfo.Infos {
			unlockhashes = append(unlockhashes, paymentInfo.EscrowAddress)
		}
	}
	return unlockhashes, nil
}
