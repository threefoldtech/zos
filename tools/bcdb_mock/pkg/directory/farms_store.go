package directory

import (
	"context"

	"net/http"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	directory "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FarmAPI holds farm releated handlers
type FarmAPI struct{}

type farmQuery struct {
	FarmName string
	OwnerID  int64
}

func (f *farmQuery) Parse(r *http.Request) mw.Response {
	var err error
	f.OwnerID, err = models.QueryInt(r, "owner")
	if err != nil {
		return mw.BadRequest(errors.Wrap(err, "owner should be a integer"))
	}
	f.FarmName = r.FormValue("name")
	return nil
}

// List farms
// TODO: add paging arguments
func (s *FarmAPI) List(ctx context.Context, db *mongo.Database, q farmQuery, opts ...*options.FindOptions) ([]directory.Farm, int64, error) {
	var filter directory.FarmFilter

	if q.OwnerID != 0 {
		filter = filter.WithOwner(q.OwnerID)
	}
	if len(q.FarmName) != 0 {
		filter = filter.WithName(q.FarmName)
	}

	cur, err := filter.Find(ctx, db, opts...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list farms")
	}
	defer cur.Close(ctx)
	out := []directory.Farm{}
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, errors.Wrap(err, "failed to load farm list")
	}

	count, err := filter.Count(ctx, db)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to count entries in farms collection")
	}

	return out, count, nil
}

// GetByName gets a farm by name
func (s *FarmAPI) GetByName(ctx context.Context, db *mongo.Database, name string) (directory.Farm, error) {
	var filter directory.FarmFilter
	filter = filter.WithName(name)

	return filter.Get(ctx, db)
}

// GetByID gets a farm by ID
func (s *FarmAPI) GetByID(ctx context.Context, db *mongo.Database, id int64) (directory.Farm, error) {
	var filter directory.FarmFilter
	filter = filter.WithID(schema.ID(id))

	return filter.Get(ctx, db)
}

// Add add farm to store
// TODO: support update farm information ?
func (s *FarmAPI) Add(ctx context.Context, db *mongo.Database, farm directory.Farm) (schema.ID, error) {
	return directory.FarmCreate(ctx, db, farm)
}
