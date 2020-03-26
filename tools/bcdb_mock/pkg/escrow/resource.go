package escrow

import (
	"context"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory/types"
	"go.mongodb.org/mongo-driver/mongo"
)

type (
	rsuPerFarmer map[int64]rsu

	rsuPerNode map[string]rsu

	rsu struct {
		cru int64
		sru int64
		hru int64
		mru int64
	}

	nodeSource interface {
		getNode(nodeID string) (types.Node, error)
	}

	dbNodeSource struct {
		ctx context.Context
		db  *mongo.Database
	}
)

func (db *dbNodeSource) getNode(nodeID string) (types.Node, error) {
	return types.NodeFilter{}.WithNodeID(nodeID).Get(db.ctx, db.db, false)
}

func processReservation(resData workloads.ReservationData, ns nodeSource) (rsuPerFarmer, error) {
	rsuPerNodeMap := make(rsuPerNode)
	for _, cont := range resData.Containers {
		rsuPerNodeMap[cont.NodeId] = rsuPerNodeMap[cont.NodeId].add(processContainer(cont))
	}
	for _, vol := range resData.Volumes {
		rsuPerNodeMap[vol.NodeId] = rsuPerNodeMap[vol.NodeId].add(processVolume(vol))
	}
	for _, zdb := range resData.Zdbs {
		rsuPerNodeMap[zdb.NodeId] = rsuPerNodeMap[zdb.NodeId].add(processZdb(zdb))
	}
	for _, k8s := range resData.Kubernetes {
		rsuPerNodeMap[k8s.NodeId] = rsuPerNodeMap[k8s.NodeId].add(processKubernetes(k8s))
	}
	rsuPerFarmerMap := make(rsuPerFarmer)
	// TODO
	for node, rsu := range rsuPerNodeMap {
		node, err := ns.getNode(node)
		if err != nil {
			return nil, errors.Wrap(err, "could not get node")
		}
		rsuPerFarmerMap[node.FarmId] = rsuPerFarmerMap[node.FarmId].add(rsu)
	}
	return rsuPerFarmerMap, nil
}

func processContainer(cont workloads.Container) rsu {
	// TODO implement after capcity field is added on Container
	return rsu{}
}

func processVolume(vol workloads.Volume) rsu {
	switch vol.Type {
	case workloads.VolumeTypeHDD:
		return rsu{
			hru: vol.Size,
		}
	case workloads.VolumeTypeSSD:
		return rsu{
			sru: vol.Size,
		}
	}
	return rsu{}
}

func processZdb(zdb workloads.ZDB) rsu {
	switch zdb.DiskType {
	case workloads.DiskTypeHDD:
		return rsu{
			hru: zdb.Size,
		}
	case workloads.DiskTypeSSD:
		return rsu{
			sru: zdb.Size,
		}
	}
	return rsu{}

}

func processKubernetes(k8s workloads.K8S) rsu {
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
