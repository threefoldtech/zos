package directory

import (
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"go.mongodb.org/mongo-driver/bson"
)

//Farm mongo db wrapper for generated TfgridDirectoryFarm
type Farm generated.TfgridDirectoryFarm1

type FarmFilter bson.D
