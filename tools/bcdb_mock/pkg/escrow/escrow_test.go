package escrow

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	rivtypes "github.com/threefoldtech/rivine/types"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	directorytypes "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/escrow/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/tfchain"

	"go.mongodb.org/mongo-driver/mongo"
)

type (
	farmAPIMock struct{}

	mockNodeSource struct{}
)

func (mns *mockNodeSource) getNode(nodeID string) (directorytypes.Node, error) {
	idInt, err := strconv.Atoi(nodeID)
	if err != nil {
		return directorytypes.Node{}, errors.New("node not found")
	}
	return directorytypes.Node{
		ID:     schema.ID(idInt),
		NodeId: nodeID,
		FarmId: int64(idInt),
	}, nil
}

const precision = 1e9

// GetByID mock method for testing purposes.
// Creates a farm and assigns some values which in some cases might be falsy in order to test the logic
func (api farmAPIMock) GetByID(ctx context.Context, db *mongo.Database, id int64) (directorytypes.Farm, error) {
	farm := directorytypes.Farm{}
	rsuPrices := []directory.TfgridDirectoryNodeResourcePrice1{{
		Cru:      5,
		Sru:      10,
		Hru:      5,
		Mru:      10,
		Nru:      0,
		Currency: directory.TfgridDirectoryNodeResourcePrice1CurrencyTFT,
	}}
	farm.ThreebotId = 12
	farm.ID = schema.ID(id)

	switch id {
	case 1:
		farm.ResourcePrices = rsuPrices
		return farm, nil
	case 2:
		return farm, nil
	case 3:
		farm.ResourcePrices = rsuPrices
		return farm, nil
	case 4:
		rsuPrices[0].Cru = -11
		farm.ResourcePrices = rsuPrices
		return farm, nil
	case 5:
		rsuPrices[0].Sru = -554564
		farm.ResourcePrices = rsuPrices
		return farm, nil
	case 6:
		rsuPrices[0].Hru = -87455
		farm.ResourcePrices = rsuPrices
		return farm, nil
	case 7:
		rsuPrices[0].Mru = -6545
		farm.ResourcePrices = rsuPrices
		return farm, nil
	}

	return directorytypes.Farm{}, fmt.Errorf("failed to get farm with id: %d", id)
}

func TestCalculateReservationCost(t *testing.T) {
	data := workloads.TfgridWorkloadsReservationData1{
		Containers: []workloads.TfgridWorkloadsReservationContainer1{
			{
				NodeId: "1",
				// TODO when capacity field is added
			},
			{
				NodeId: "1",
				// TODO when capacity field is added
			},
			{
				NodeId: "3",
				// TODO when capacity field is added
			},
			{
				NodeId: "3",
				// TODO when capacity field is added
			},
			{
				NodeId: "3",
				// TODO when capacity field is added
			},
			{
				NodeId: "3",
				// TODO when capacity field is added
			},
		},
		Volumes: []workloads.TfgridWorkloadsReservationVolume1{
			{
				NodeId: "1",
				Type:   workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:   500,
			},
			{
				NodeId: "1",
				Type:   workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:   500,
			},
			{
				NodeId: "3",
				Type:   workloads.TfgridWorkloadsReservationVolume1TypeSSD,
				Size:   100,
			},
			{
				NodeId: "3",
				Type:   workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:   2500,
			},
			{
				NodeId: "3",
				Type:   workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:   1000,
			},
		},
		Zdbs: []workloads.TfgridWorkloadsReservationZdb1{
			{
				NodeId:   "1",
				DiskType: workloads.TfgridWorkloadsReservationZdb1DiskTypeSsd,
				Size:     750,
			},
			{
				NodeId:   "3",
				DiskType: workloads.TfgridWorkloadsReservationZdb1DiskTypeSsd,
				Size:     250,
			},
			{
				NodeId:   "3",
				DiskType: workloads.TfgridWorkloadsReservationZdb1DiskTypeHdd,
				Size:     500,
			},
		},
		Kubernetes: []workloads.TfgridWorkloadsReservationK8S1{
			{
				NodeId: "1",
				Size:   1,
			},
			{
				NodeId: "1",
				Size:   2,
			},
			{
				NodeId: "1",
				Size:   2,
			},
			{
				NodeId: "3",
				Size:   2,
			},
			{
				NodeId: "3",
				Size:   2,
			},
			{
				NodeId: "3",
				Size:   2,
			},
		},
	}

	farmRsu, err := processReservation(data, &mockNodeSource{})
	assert.NoError(t, err)

	escrow := Escrow{
		wallet:             tfchain.Wallet{},
		db:                 nil,
		reservationChannel: nil,
		farmAPI:            farmAPIMock{},
	}

	res, err := escrow.CalculateReservationCost(farmRsu)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	assert.True(t, len(res) == 2)
	assert.Equal(t, types.Currency{Currency: rivtypes.NewCurrency64(15125 * precision)}, res[1])
	assert.Equal(t, types.Currency{Currency: rivtypes.NewCurrency64(26650 * precision)}, res[3])
}

func TestCalculateReservationCostForUnknownFarmer(t *testing.T) {
	data := makeMockReservationData("15")

	farmRsu, err := processReservation(data, &mockNodeSource{})
	assert.NoError(t, err)

	escrow := Escrow{
		wallet:             tfchain.Wallet{},
		db:                 nil,
		reservationChannel: nil,
		farmAPI:            farmAPIMock{},
	}

	_, err = escrow.CalculateReservationCost(farmRsu)
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestCalculateReservationCostForFarmerWithoutPrices(t *testing.T) {
	data := makeMockReservationData("2")

	farmRsu, err := processReservation(data, &mockNodeSource{})
	assert.NoError(t, err)

	escrow := Escrow{
		wallet:             tfchain.Wallet{},
		db:                 nil,
		reservationChannel: nil,
		farmAPI:            farmAPIMock{},
	}

	_, err = escrow.CalculateReservationCost(farmRsu)
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestCalculateReservationCostForFarmerWithFalsyCruPrice(t *testing.T) {
	data := makeMockReservationData("4")

	farmRsu, err := processReservation(data, &mockNodeSource{})
	assert.NoError(t, err)

	escrow := Escrow{
		wallet:             tfchain.Wallet{},
		db:                 nil,
		reservationChannel: nil,
		farmAPI:            farmAPIMock{},
	}

	_, err = escrow.CalculateReservationCost(farmRsu)
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestCalculateReservationCostForFarmerWithFalsySruPrice(t *testing.T) {
	data := makeMockReservationData("5")

	farmRsu, err := processReservation(data, &mockNodeSource{})
	assert.NoError(t, err)

	escrow := Escrow{
		wallet:             tfchain.Wallet{},
		db:                 nil,
		reservationChannel: nil,
		farmAPI:            farmAPIMock{},
	}

	_, err = escrow.CalculateReservationCost(farmRsu)
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestCalculateReservationCostForFarmerWithFalsyHruPrice(t *testing.T) {
	data := makeMockReservationData("6")

	farmRsu, err := processReservation(data, &mockNodeSource{})
	assert.NoError(t, err)

	escrow := Escrow{
		wallet:             tfchain.Wallet{},
		db:                 nil,
		reservationChannel: nil,
		farmAPI:            farmAPIMock{},
	}

	_, err = escrow.CalculateReservationCost(farmRsu)
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestCalculateReservationCostForFarmerWithFalsyMruPrice(t *testing.T) {
	data := makeMockReservationData("7")

	farmRsu, err := processReservation(data, &mockNodeSource{})
	assert.NoError(t, err)

	escrow := Escrow{
		wallet:             tfchain.Wallet{},
		db:                 nil,
		reservationChannel: nil,
		farmAPI:            farmAPIMock{},
	}

	_, err = escrow.CalculateReservationCost(farmRsu)
	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func makeMockReservationData(id string) workloads.TfgridWorkloadsReservationData1 {
	return workloads.TfgridWorkloadsReservationData1{
		Containers: []workloads.TfgridWorkloadsReservationContainer1{
			{
				NodeId: id,
				// TODO when capacity field is added
			},
		},
		Volumes: []workloads.TfgridWorkloadsReservationVolume1{
			{
				NodeId: id,
				Type:   workloads.TfgridWorkloadsReservationVolume1TypeHDD,
				Size:   500,
			},
		},
		Zdbs: []workloads.TfgridWorkloadsReservationZdb1{
			{
				NodeId:   id,
				DiskType: workloads.TfgridWorkloadsReservationZdb1DiskTypeSsd,
				Size:     750,
			},
		},
		Kubernetes: []workloads.TfgridWorkloadsReservationK8S1{
			{
				NodeId: id,
				Size:   1,
			},
		},
	}
}
