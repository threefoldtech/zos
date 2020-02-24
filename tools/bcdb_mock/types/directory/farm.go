package directory

import (
	"context"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	farmCollection = "farm"
)

//Farm mongo db wrapper for generated TfgridDirectoryFarm
type Farm generated.TfgridDirectoryFarm1

// FarmFilter type
type FarmFilter bson.D

// WithID filter farm with ID
func (f FarmFilter) WithID(id schema.ID) FarmFilter {
	return append(f, bson.E{Key: "_id", Value: id})
}

// WithName filter farm with name
func (f FarmFilter) WithName(name string) FarmFilter {
	return append(f, bson.E{Key: "name", Value: name})
}

// Find run the filter and return a cursor result
func (f FarmFilter) Find(ctx context.Context, db *mongo.Database, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	col := db.Collection(farmCollection)
	if f == nil {
		f = FarmFilter{}
	}
	return col.Find(ctx, f, opts...)
}

// Get one farm that matches the filter
func (f FarmFilter) Get(ctx context.Context, db *mongo.Database) (farm Farm, err error) {
	if f == nil {
		f = FarmFilter{}
	}
	col := db.Collection(farmCollection)
	result := col.FindOne(ctx, f, options.FindOne())

	err = result.Err()
	if err != nil {
		return
	}

	err = result.Decode(&farm)
	return
}

// FarmCreate creates a new farm
func FarmCreate(ctx context.Context, db *mongo.Database, farm Farm) (schema.ID, error) {
	col := db.Collection(farmCollection)
	id, err := models.NextID(ctx, db, farmCollection)
	if err != nil {
		return id, err
	}

	farm.ID = id
	_, err = col.InsertOne(ctx, farm)
	return id, err
}

// Setup sets up indexes for types, must be called at least
// Onetime during the life time of the object
func Setup(ctx context.Context, db *mongo.Database) error {
	farm := db.Collection(farmCollection)
	_, err := farm.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.M{"name": 1},
			Options: options.Index().SetUnique(true),
		},
	})

	return err
}
