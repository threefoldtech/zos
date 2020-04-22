package builders

import (
	"encoding/json"
	"io"

	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

// ZdbBuilder is a struct that can build ZDB's
type ZdbBuilder struct {
	workloads.ZDB
}

// NewZdbBuilder creates a new zdb builder and initializes some default values
func NewZdbBuilder(nodeID string, size int64, mode workloads.ZDBModeEnum, diskType workloads.DiskTypeEnum) *ZdbBuilder {
	return &ZdbBuilder{
		ZDB: workloads.ZDB{
			NodeId:   nodeID,
			Size:     size,
			Mode:     mode,
			DiskType: diskType,
		},
	}
}

// LoadZdbBuilder loads a zdb builder based on a file path
func LoadZdbBuilder(reader io.Reader) (*ZdbBuilder, error) {
	zdb := workloads.ZDB{}

	err := json.NewDecoder(reader).Decode(&zdb)
	if err != nil {
		return &ZdbBuilder{}, err
	}

	return &ZdbBuilder{ZDB: zdb}, nil
}

// Save saves the zdb builder to an IO.Writer
func (z *ZdbBuilder) Save(writer io.Writer) error {
	err := json.NewEncoder(writer).Encode(z.ZDB)
	if err != nil {
		return err
	}
	return err
}

// Build validates and encrypts the zdb secret
func (z *ZdbBuilder) Build() (workloads.ZDB, error) {
	encrypted, err := encryptSecret(z.ZDB.Password, z.ZDB.NodeId)
	if err != nil {
		return workloads.ZDB{}, err
	}

	z.ZDB.Password = encrypted
	return z.ZDB, nil
}

// WithNodeID sets the node ID to the zdb
func (z *ZdbBuilder) WithNodeID(nodeID string) *ZdbBuilder {
	z.ZDB.NodeId = nodeID
	return z
}

// WithSize sets the size on the zdb
func (z *ZdbBuilder) WithSize(size int64) *ZdbBuilder {
	z.ZDB.Size = size
	return z
}

// WithMode sets the mode to the zdb
func (z *ZdbBuilder) WithMode(mode workloads.ZDBModeEnum) *ZdbBuilder {
	z.ZDB.Mode = mode
	return z
}

// WithPassword sets the password to the zdb
func (z *ZdbBuilder) WithPassword(password string) *ZdbBuilder {
	z.ZDB.Password = password
	return z
}

// WithDiskType sets the disktype to the zdb
func (z *ZdbBuilder) WithDiskType(diskType workloads.DiskTypeEnum) *ZdbBuilder {
	z.ZDB.DiskType = diskType
	return z
}

// WithPublic sets if public to the zdb
func (z *ZdbBuilder) WithPublic(public bool) *ZdbBuilder {
	z.ZDB.Public = public
	return z
}

// WithStatsAggregator sets the stats aggregators to the zdb
func (z *ZdbBuilder) WithStatsAggregator(aggregators []workloads.StatsAggregator) *ZdbBuilder {
	z.ZDB.StatsAggregator = aggregators
	return z
}
