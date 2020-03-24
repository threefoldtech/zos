package escrow

import "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"

type (
	rsuPerFarmer map[int64]rsu

	rsu struct {
		cru int64
		sru int64
		hru int64
		mru int64
	}
)

func processReservation(resData workloads.TfgridWorkloadsReservationData1) rsuPerFarmer {
	rsuPerFarmerMap := make(rsuPerFarmer)
	for _, cont := range resData.Containers {
		rsuPerFarmerMap[cont.FarmerTid] = rsuPerFarmerMap[cont.FarmerTid].add(processContainer(cont))
	}
	for _, vol := range resData.Volumes {
		rsuPerFarmerMap[vol.FarmerTid] = rsuPerFarmerMap[vol.FarmerTid].add(processVolume(vol))
	}
	for _, zdb := range resData.Zdbs {
		rsuPerFarmerMap[zdb.FarmerTid] = rsuPerFarmerMap[zdb.FarmerTid].add(processZdb(zdb))
	}
	for _, k8s := range resData.Kubernetes {
		rsuPerFarmerMap[k8s.FarmerTid] = rsuPerFarmerMap[k8s.FarmerTid].add(processKubernetes(k8s))
	}
	return rsuPerFarmerMap
}

func processContainer(cont workloads.TfgridWorkloadsReservationContainer1) rsu {
	// TODO implement after capcity field is added on TfgridWorkloadsReservationContainer1
	return rsu{}
}

func processVolume(vol workloads.TfgridWorkloadsReservationVolume1) rsu {
	switch vol.Type {
	case workloads.TfgridWorkloadsReservationVolume1TypeHDD:
		return rsu{
			hru: vol.Size,
		}
	case workloads.TfgridWorkloadsReservationVolume1TypeSSD:
		return rsu{
			sru: vol.Size,
		}
	}
	return rsu{}
}

func processZdb(zdb workloads.TfgridWorkloadsReservationZdb1) rsu {
	switch zdb.DiskType {
	case workloads.TfgridWorkloadsReservationZdb1DiskTypeHdd:
		return rsu{
			hru: zdb.Size,
		}
	case workloads.TfgridWorkloadsReservationZdb1DiskTypeSsd:
		return rsu{
			sru: zdb.Size,
		}
	}
	return rsu{}

}

func processKubernetes(k8s workloads.TfgridWorkloadsReservationK8S1) rsu {
	switch k8s.Size {
	case 1:
		return rsu{
			cru: 1,
			mru: 2,
			sru: 50,
		}
	case 2:
		return rsu{
			cru: 2,
			mru: 4,
			sru: 100,
		}
	}
	return rsu{}

}

func (r rsu) add(other rsu) rsu {
	return rsu{
		cru: r.cru + other.cru,
		sru: r.sru + other.sru,
		hru: r.hru + other.hru,
		mru: r.mru + other.mru,
	}
}
