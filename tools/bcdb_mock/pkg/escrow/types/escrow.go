package types

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	rivtypes "github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/pkg/schema"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
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
	// ReservationPaymentInformation stores the reservation payment information
	ReservationPaymentInformation struct {
		ReservationID schema.ID   `bson:"_id"`
		Expiration    schema.Date `bson:"expiration"`
		Infos         []info      `bson:"infos"`
		Paid          bool        `bson:"paid"`
	}

	info struct {
		FarmerID      schema.ID `bson:"farmer_id"`
		TotalAmount   Currency  `bson:"total_amount"`
		EscrowAddress Address   `bson:"escrow_address"`
	}

	// Currency is an amount of tokens
	Currency struct {
		rivtypes.Currency
	}

	// Address is an on chain address
	Address struct {
		rivtypes.UnlockHash
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
	filter := bson.M{"paid": false, "expiration": bson.M{"$gt": schema.Date{Time: time.Now()}}}
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

// GetAllAddresses gets all in use addresses from the escrowcollections
func GetAllAddresses(ctx context.Context, db *mongo.Database) ([]Address, error) {
	cursor, err := db.Collection(EscrowCollection).Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create cursor over payment infos")
	}
	defer cursor.Close(ctx)
	unlockhashes := make([]Address, 0)
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

// MarshalBSONValue implements bson.ValueMarshaler interface
func (c *Currency) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(c.String())
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler interface
func (c *Currency) UnmarshalBSONValue(t bsontype.Type, raw []byte) error {
	if t != bsontype.String {
		return errors.New("expected currency to be saved in bson as a string")
	}

	rv := bson.RawValue{Type: t, Value: raw}
	text := rv.StringValue()
	_, err := fmt.Sscan(text, c)
	return err
}

// MarshalBSONValue implements bson.ValueMarshaler interface
func (a *Address) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(a.String())
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler interface
func (a *Address) UnmarshalBSONValue(t bsontype.Type, raw []byte) error {
	if t != bsontype.String {
		return errors.New("expected address to be saved in bson as a string")
	}

	rv := bson.RawValue{Type: t, Value: raw}
	text := rv.StringValue()
	return a.LoadString(text)
}
