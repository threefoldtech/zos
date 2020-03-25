package escrow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	rivtypes "github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	directorytypes "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/tfchain"

	"go.mongodb.org/mongo-driver/mongo"
)

type farmApiMock struct{}

const precision = 1e9

func (api farmApiMock) GetByID(ctx context.Context, db *mongo.Database, id int64) (directorytypes.Farm, error) {
	farm1 := directorytypes.Farm{}
	farm1.ThreebotId = 12
	farm1.ID = schema.ID(id)
	farm1.ResourcePrices = []directory.TfgridDirectoryNodeResourcePrice1{{
		Cru:      5,
		Sru:      10,
		Hru:      5,
		Mru:      10,
		Nru:      0,
		Currency: directory.TfgridDirectoryNodeResourcePrice1CurrencyTFT,
	}}
	return farm1, nil
}

func TestCalculateReservationCost(t *testing.T) {
	data := workloads.TfgridWorkloadsReservationData1{
		Containers: []workloads.TfgridWorkloadsReservationContainer1{
			{
				FarmerTid: 1,
				// TODO when capacity field is added
			},
			{
				FarmerTid: 1,
				// TODO when capacity field is added
			},
			{
				FarmerTid: 3,
				// TODO when capacity field is added
			},
			{
				FarmerTid: 3,
				// TODO when capacity field is added
			},
			{
				FarmerTid: 3,
				// TODO when capacity field is added
			},
			{
				FarmerTid: 3,
				// TODO when capacity field is added
			},
		},
		Volumes: []workloads.TfgridWorkloadsReservationVolume1{
			{
				FarmerTid: 1,
				Type:      workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:      500,
			},
			{
				FarmerTid: 1,
				Type:      workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:      500,
			},
			{
				FarmerTid: 3,
				Type:      workloads.TfgridWorkloadsReservationVolume1TypeSSD,
				Size:      100,
			},
			{
				FarmerTid: 3,
				Type:      workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:      2500,
			},
			{
				FarmerTid: 3,
				Type:      workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:      1000,
			},
		},
		Zdbs: []workloads.TfgridWorkloadsReservationZdb1{
			{
				FarmerTid: 1,
				DiskType:  workloads.TfgridWorkloadsReservationZdb1DiskTypeSsd,
				Size:      750,
			},
			{
				FarmerTid: 3,
				DiskType:  workloads.TfgridWorkloadsReservationZdb1DiskTypeSsd,
				Size:      250,
			},
			{
				FarmerTid: 3,
				DiskType:  workloads.TfgridWorkloadsReservationZdb1DiskTypeHdd,
				Size:      500,
			},
		},
		Kubernetes: []workloads.TfgridWorkloadsReservationK8S1{
			{
				FarmerTid: 1,
				Size:      1,
			},
			{
				FarmerTid: 1,
				Size:      2,
			},
			{
				FarmerTid: 1,
				Size:      2,
			},
			{
				FarmerTid: 3,
				Size:      2,
			},
			{
				FarmerTid: 3,
				Size:      2,
			},
			{
				FarmerTid: 3,
				Size:      2,
			},
		},
	}

	farmRsu := processReservation(data)

	escrow := Escrow{
		wallet:             tfchain.Wallet{},
		db:                 nil,
		reservationChannel: nil,
		farmApi:            farmApiMock{},
	}

	res, err := escrow.CalculateReservationCost(farmRsu)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	assert.True(t, len(res) == 2)
	assert.Equal(t, rivtypes.NewCurrency64(15125*precision), res[1])
	assert.Equal(t, rivtypes.NewCurrency64(26650*precision), res[3])
}
