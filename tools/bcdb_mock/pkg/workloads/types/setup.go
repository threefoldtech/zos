package types

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// Setup sets up indexes for types, must be called at least
// Onetime during the life time of the object
func Setup(ctx context.Context, db *mongo.Database) error {
	return nil
}
