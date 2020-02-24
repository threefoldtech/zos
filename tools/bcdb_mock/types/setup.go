package types

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/tools/bcdb_mock/types/directory"
	"go.mongodb.org/mongo-driver/mongo"
)

// Setup call is needed to make sure indexes are setup and working before
// server is started
func Setup(ctx context.Context, db *mongo.Database) {
	schemas := []func(context.Context, *mongo.Database) error{
		directory.Setup,
	}

	for _, setup := range schemas {
		if err := setup(ctx, db); err != nil {
			log.Error().Err(err).Msg("failed to setup index")
		}
	}
}
