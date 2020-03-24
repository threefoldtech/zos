package escrow

import (
	"testing"

	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
)

func TestProcessReservation(t *testing.T) {
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
				FarmerTid: 2,
				// TODO when capacity field is added
			},
			{
				FarmerTid: 2,
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
				FarmerTid: 2,
				Type:      workloads.TfgridWorkloadsReservationVolume1TypeSSD,
				Size:      100,
			},
			{
				FarmerTid: 2,
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
				FarmerTid: 2,
				Size:      2,
			},
			{
				FarmerTid: 2,
				Size:      2,
			},
			{
				FarmerTid: 3,
				Size:      2,
			},
		},
	}

	farmRsu := processReservation(data)

	if len(farmRsu) != 3 {
		t.Errorf("Found %d farmers, expected to find 3", len(farmRsu))
	}

	// TODO: Update when container capacity field is here
	// check farm tid 1
	rsu := farmRsu[1]
	if rsu.cru != 5 {
		t.Errorf("Farmer 1 total cru is %d, expected 5", rsu.cru)
	}
	if rsu.mru != 10 {
		t.Errorf("Farmer 1 total mru is %d, expected 10", rsu.mru)
	}
	if rsu.sru != 1000 {
		t.Errorf("Farmer 1 total sru is %d, expected 1000", rsu.sru)
	}
	if rsu.hru != 1000 {
		t.Errorf("Farmer 1 total hru is %d, expected 1000", rsu.hru)
	}

	// check farm tid 2
	rsu = farmRsu[2]
	if rsu.cru != 4 {
		t.Errorf("Farmer 2 total cru is %d, expected 4", rsu.cru)
	}
	if rsu.mru != 8 {
		t.Errorf("Farmer 2 total mru is %d, expected 8", rsu.mru)
	}
	if rsu.sru != 300 {
		t.Errorf("Farmer 2 total sru is %d, expected 300", rsu.sru)
	}
	if rsu.hru != 2500 {
		t.Errorf("Farmer 2 total hru is %d, expected 2500", rsu.hru)
	}

	// check farm tid 3
	rsu = farmRsu[3]
	if rsu.cru != 2 {
		t.Errorf("Farmer 3 total cru is %d, expected 2", rsu.cru)
	}
	if rsu.mru != 4 {
		t.Errorf("Farmer 3 total mru is %d, expected 4", rsu.mru)
	}
	if rsu.sru != 350 {
		t.Errorf("Farmer 3 total sru is %d, expected 350", rsu.sru)
	}
	if rsu.hru != 1500 {
		t.Errorf("Farmer 3 total hru is %d, expected 1500", rsu.hru)
	}
}

func TestProcessContainer(t *testing.T) {
	// TODO once capacity field is added on container
}

func TestProcessVolume(t *testing.T) {
	testSize := int64(27) // can be random number

	vol := workloads.TfgridWorkloadsReservationVolume1{
		Size: testSize,
		Type: workloads.TfgridWorkloadsReservationVolume1TypeHDD,
	}
	rsu := processVolume(vol)

	if rsu.cru != 0 {
		t.Errorf("Processed volume cru is %d, expected 0", rsu.cru)
	}
	if rsu.mru != 0 {
		t.Errorf("Processed volume mru is %d, expected 0", rsu.mru)
	}
	if rsu.sru != 0 {
		t.Errorf("Processed volume sru is %d, expected 0", rsu.sru)
	}
	if rsu.hru != testSize {
		t.Errorf("Processed volume hru is %d, expected %d", rsu.hru, testSize)
	}

	vol = workloads.TfgridWorkloadsReservationVolume1{
		Size: testSize,
		Type: workloads.TfgridWorkloadsReservationVolume1TypeSSD,
	}
	rsu = processVolume(vol)

	if rsu.cru != 0 {
		t.Errorf("Processed volume cru is %d, expected 0", rsu.cru)
	}
	if rsu.mru != 0 {
		t.Errorf("Processed volume mru is %d, expected 0", rsu.mru)
	}
	if rsu.sru != testSize {
		t.Errorf("Processed volume sru is %d, expected %d", rsu.sru, testSize)
	}
	if rsu.hru != 0 {
		t.Errorf("Processed volume hru is %d, expected 0", rsu.hru)
	}
}

func TestProcessZdb(t *testing.T) {
	testSize := int64(43) // can be random number

	zdb := workloads.TfgridWorkloadsReservationZdb1{
		DiskType: workloads.TfgridWorkloadsReservationZdb1DiskTypeHdd,
		Size:     testSize,
	}
	rsu := processZdb(zdb)

	if rsu.cru != 0 {
		t.Errorf("Processed zdb cru is %d, expected 0", rsu.cru)
	}
	if rsu.mru != 0 {
		t.Errorf("Processed zdb mru is %d, expected 0", rsu.mru)
	}
	if rsu.sru != 0 {
		t.Errorf("Processed zdb sru is %d, expected 0", rsu.sru)
	}
	if rsu.hru != testSize {
		t.Errorf("Processed zdb hru is %d, expected %d", rsu.hru, testSize)
	}

	zdb = workloads.TfgridWorkloadsReservationZdb1{
		DiskType: workloads.TfgridWorkloadsReservationZdb1DiskTypeSsd,
		Size:     testSize,
	}
	rsu = processZdb(zdb)

	if rsu.cru != 0 {
		t.Errorf("Processed zdb cru is %d, expected 0", rsu.cru)
	}
	if rsu.mru != 0 {
		t.Errorf("Processed zdb mru is %d, expected 0", rsu.mru)
	}
	if rsu.sru != testSize {
		t.Errorf("Processed zdb sru is %d, expected %d", rsu.sru, testSize)
	}
	if rsu.hru != 0 {
		t.Errorf("Processed zdb hru is %d, expected 0", rsu.hru)
	}
}

func TestProcessKubernetes(t *testing.T) {
	k8s := workloads.TfgridWorkloadsReservationK8S1{
		Size: 1,
	}
	rsu := processKubernetes(k8s)

	if rsu.cru != 1 {
		t.Errorf("Processed zdb cru is %d, expected 1", rsu.cru)
	}
	if rsu.mru != 2 {
		t.Errorf("Processed zdb mru is %d, expected 2", rsu.mru)
	}
	if rsu.sru != 50 {
		t.Errorf("Processed zdb sru is %d, expected 50", rsu.sru)
	}
	if rsu.hru != 0 {
		t.Errorf("Processed zdb hru is %d, expected 0", rsu.hru)
	}

	k8s = workloads.TfgridWorkloadsReservationK8S1{
		Size: 2,
	}
	rsu = processKubernetes(k8s)

	if rsu.cru != 2 {
		t.Errorf("Processed zdb cru is %d, expected 2", rsu.cru)
	}
	if rsu.mru != 4 {
		t.Errorf("Processed zdb mru is %d, expected 4", rsu.mru)
	}
	if rsu.sru != 100 {
		t.Errorf("Processed zdb sru is %d, expected 100", rsu.sru)
	}
	if rsu.hru != 0 {
		t.Errorf("Processed zdb hru is %d, expected 0", rsu.hru)
	}
}

func TestRsuAdd(t *testing.T) {
	first := rsu{cru: 1, mru: 2, sru: 3, hru: 4}
	second := rsu{cru: 8, mru: 6, sru: 4, hru: 2}
	result := first.add(second)

	if result.cru != 9 {
		t.Errorf("Result cru is %d, expected 9", result.cru)
	}
	if result.mru != 8 {
		t.Errorf("Result mru is %d, expected 8", result.mru)
	}
	if result.sru != 7 {
		t.Errorf("Result sru is %d, expected 7", result.sru)
	}
	if result.hru != 6 {
		t.Errorf("Result hru is %d, expected 6", result.hru)
	}
}
