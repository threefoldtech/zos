package storage

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

const (
	testType1         = gridtypes.WorkloadType("type1")
	testType2         = gridtypes.WorkloadType("type2")
	testSharableType1 = gridtypes.WorkloadType("sharable1")
)

type TestData struct{}

func (t TestData) Valid(getter gridtypes.WorkloadGetter) error {
	return nil
}

func (t TestData) Challenge(w io.Writer) error {
	return nil
}

func (t TestData) Capacity() (gridtypes.Capacity, error) {
	return gridtypes.Capacity{}, nil
}

func init() {
	gridtypes.RegisterType(testType1, TestData{})
	gridtypes.RegisterType(testType2, TestData{})
	gridtypes.RegisterSharableType(testSharableType1, TestData{})
}

func TestCreateDeployment(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
	}
	err = db.Create(&dl)
	require.NoError(err)

	err = db.Create(&dl)
	require.ErrorIs(err, provision.ErrDeploymentExists)
}

func TestAddWorkload(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	db, err := New(path)
	require.NoError(err)

	err = db.Add(1, 10, "vm1", testType1, false)
	require.ErrorIs(err, provision.ErrDeploymentNotExists)

	dl := gridtypes.Deployment{
		Version:     1,
		TwinID:      1,
		ContractID:  10,
		Description: "description",
		Metadata:    "some metadata",
	}

	err = db.Create(&dl)
	require.NoError(err)

	err = db.Add(1, 10, "vm1", testType1, false)
	require.NoError(err)

	err = db.Add(1, 10, "vm1", testType1, false)
	require.ErrorIs(err, provision.ErrWorkloadExists)

}
