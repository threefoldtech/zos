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
	// ErrEscrowExists returned if escrow with same name exists
	ErrEscrowExists = errors.New("escrow with same name or email exists")
	// ErrEscrowNotFound is returned if escrow is not found
	ErrEscrowNotFound = errors.New("escrow not found")
	// ErrAuthorization returned if escrow is not allowed to do an operation
	ErrAuthorization = errors.New("operation not allowed")
)

type (
	ReservationPaymentInformation struct {
		reservationID schema.ID   `bson:"_id"`
		expiration    schema.Date `bson:"expiration"`
		infos         []info      `bson:"infos"`
		paid          bool        `bson:"paid"`
	}
	info struct {
		farmerID      schema.ID           `bson:"farmer_id"`
		totalAmount   rivtypes.Currency   `bson:"total_amount"`
		escrowAddress rivtypes.UnlockHash `bson:"escrow_address"`
	}
)

// ReservationFilter type
type ReservationFilter bson.D

// WithID filters reservation payment information with ID
func (f ReservationFilter) WithID(id schema.ID) ReservationFilter {
	if id == 0 {
		return f
	}
	return append(f, bson.E{Key: "_id", Value: id})
}

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
	filter := bson.M{"_id": update.reservationID}
	// actually update the user with final data
	if _, err := db.Collection(EscrowCollection).UpdateOne(ctx, filter, bson.M{"$set": update}); err != nil {
		return err
	}

	return nil
}

// GetAllActiveReservationPaymentInfos get all active reservation payment information
func GetAllActiveReservationPaymentInfos(ctx context.Context, db *mongo.Database) ([]ReservationPaymentInformation, error) {
	filter := bson.M{"paid": false, "expiration": bson.M{"$lt": time.Now()}}
	cursor, err := db.Collection(EscrowCollection).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	paymentInfos := make([]ReservationPaymentInformation, 0)
	err = cursor.All(ctx, &paymentInfos)
	return paymentInfos, err
}

func GetAllAddresses(ctx context.Context, db *mongo.Database) ([]rivtypes.UnlockHash, error) {
	cursor, err := db.Collection(EscrowCollection).Find(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	unlockhashes := make([]rivtypes.UnlockHash, 0)
	var reservationPaymentInfo ReservationPaymentInformation
	for cursor.Next(ctx) {
		err = cursor.Decode(&reservationPaymentInfo)
		if err != nil {
			return nil, err
		}
		for _, paymentInfo := range reservationPaymentInfo.infos {
			unlockhashes = append(unlockhashes, paymentInfo.escrowAddress)
		}
	}
	return unlockhashes, nil
}
