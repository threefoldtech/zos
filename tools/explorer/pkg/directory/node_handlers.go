package directory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zaibon/httpsig"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models"
	generated "github.com/threefoldtech/zos/tools/explorer/models/generated/directory"
	"github.com/threefoldtech/zos/tools/explorer/mw"
	"github.com/threefoldtech/zos/tools/explorer/pkg/directory/types"
	directory "github.com/threefoldtech/zos/tools/explorer/pkg/directory/types"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
)

type nodesSpecs struct {
	AmountRegisteredNodes int64   `json:"amountOfRegisteredNodes"`
	OnlineNodes           int     `json:"onlineNodes"`
	Countries             int     `json:"countries"`
	TotalCru              uint64  `json:"totalCru"`
	TotalSru              float64 `json:"totalSru"`
	TotalHru              float64 `json:"totalHru"`
	TotalMru              float64 `json:"totalMru"`
	Networks              uint16  `json:"networks"`
	Volumes               uint16  `json:"volumes"`
	Containers            uint16  `json:"containers"`
	Zdbs                  uint16  `json:"zdbs"`
	K8s                   uint16  `json:"k8s"`
}

func (s *NodeAPI) registerNode(r *http.Request) (interface{}, mw.Response) {
	log.Info().Msg("node register request received")

	defer r.Body.Close()

	var n directory.Node
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		return nil, mw.BadRequest(err)
	}

	//make sure node can not set public config
	n.PublicConfig = nil
	db := mw.Database(r)
	if _, err := s.Add(r.Context(), db, n); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, mw.NotFound(fmt.Errorf("farm with id:%d does not exists", n.FarmId))
		}
		return nil, mw.Error(err)
	}

	log.Info().Msgf("node registered: %+v\n", n)

	return nil, mw.Created()
}

func (s *NodeAPI) nodeDetail(r *http.Request) (interface{}, mw.Response) {
	nodeID := mux.Vars(r)["node_id"]
	q := nodeQuery{}
	if err := q.Parse(r); err != nil {
		return nil, err
	}
	db := mw.Database(r)

	node, err := s.Get(r.Context(), db, nodeID, q.Proofs)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	return node, nil
}

func (s *NodeAPI) listNodes(r *http.Request) (interface{}, mw.Response) {
	q := nodeQuery{}
	if err := q.Parse(r); err != nil {
		return nil, err
	}

	db := mw.Database(r)
	pager := models.PageFromRequest(r)
	nodes, total, err := s.List(r.Context(), db, q, pager)
	if err != nil {
		return nil, mw.Error(err)
	}

	pages := fmt.Sprintf("%d", models.Pages(pager, total))
	return nodes, mw.Ok().WithHeader("Pages", pages)
}

func (s *NodeAPI) nodeStats(r *http.Request) (interface{}, mw.Response) {
	q := nodeQuery{}
	if err := q.Parse(r); err != nil {
		return nil, err
	}

	db := mw.Database(r)
	nodes, total, err := s.List(r.Context(), db, q, nil)
	if err != nil {
		return nil, mw.Error(err)
	}

	nodeSpecs := caculateNodeSpecs(nodes)
	nodeSpecs.AmountRegisteredNodes = total

	return nodeSpecs, mw.Ok()
}

func caculateNodeSpecs(nodes []directory.Node) nodesSpecs {
	nodeCountries := make([]string, 0)
	nodeSpecs := nodesSpecs{}
	for _, node := range nodes {
		_, found := find(nodeCountries, node.Location.Country)
		if !found {
			nodeCountries = append(nodeCountries, node.Location.Country)
			nodeSpecs.Countries++
		}
		if time.Now().Before(node.Updated.Time.Add(time.Minute * 20)) {
			nodeSpecs.OnlineNodes++
		}
		nodeSpecs.TotalCru += node.TotalResources.Cru
		nodeSpecs.TotalSru += node.TotalResources.Sru
		nodeSpecs.TotalHru += node.TotalResources.Hru
		nodeSpecs.TotalMru += node.TotalResources.Mru
		nodeSpecs.Networks += node.Workloads.Network
		nodeSpecs.Volumes += node.Workloads.Volume
		nodeSpecs.Containers += node.Workloads.Container
		nodeSpecs.Zdbs += node.Workloads.ZDBNamespace
		nodeSpecs.K8s += node.Workloads.K8sVM

	}
	return nodeSpecs
}

// find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func countOnlineNodes(nodes []directory.Node) int {
	onlinecounter := 0
	for _, node := range nodes {
		timestamp := time.Now()
		minutes := (timestamp.Sub(node.Updated.Time)) / 60
		if minutes < 20 {
			onlinecounter++
		}
	}
	return onlinecounter
}

func (s *NodeAPI) registerCapacity(r *http.Request) (interface{}, mw.Response) {
	x := struct {
		Capacity   generated.ResourceAmount `json:"capacity,omitempty"`
		DMI        dmi.DMI                  `json:"dmi,omitempty"`
		Disks      capacity.Disks           `json:"disks,omitempty"`
		Hypervisor []string                 `json:"hypervisor,omitempty"`
	}{}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
		return nil, mw.BadRequest(err)
	}

	nodeID := mux.Vars(r)["node_id"]
	hNodeID := httpsig.KeyIDFromContext(r.Context())
	if nodeID != hNodeID {
		return nil, mw.Forbidden(fmt.Errorf("trying to register capacity for nodeID %s while you are %s", nodeID, hNodeID))
	}

	db := mw.Database(r)

	if err := s.updateTotalCapacity(r.Context(), db, nodeID, x.Capacity); err != nil {
		return nil, mw.NotFound(err)
	}

	if err := s.StoreProof(r.Context(), db, nodeID, x.DMI, x.Disks, x.Hypervisor); err != nil {
		return nil, mw.Error(err)
	}

	return nil, nil
}

func (s *NodeAPI) registerIfaces(r *http.Request) (interface{}, mw.Response) {
	log.Debug().Msg("network interfaces register request received")

	defer r.Body.Close()

	var input []generated.Iface
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, mw.BadRequest(err)
	}

	nodeID := mux.Vars(r)["node_id"]
	hNodeID := httpsig.KeyIDFromContext(r.Context())
	if nodeID != hNodeID {
		return nil, mw.Forbidden(fmt.Errorf("trying to register interfaces for nodeID %s while you are %s", nodeID, hNodeID))
	}

	db := mw.Database(r)
	if err := s.SetInterfaces(r.Context(), db, nodeID, input); err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}

func (s *NodeAPI) configurePublic(r *http.Request) (interface{}, mw.Response) {
	var iface generated.PublicIface

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&iface); err != nil {
		return nil, mw.BadRequest(err)
	}

	db := mw.Database(r)
	nodeID := mux.Vars(r)["node_id"]

	node, err := s.Get(r.Context(), db, nodeID, false)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	// ensure it is the farmer that does the call
	authorized, merr := isFarmerAuthorized(r, node, db)
	if err != nil {
		return nil, merr
	}

	if !authorized {
		return nil, mw.Forbidden(fmt.Errorf("only the farmer can configured the public interface of its nodes"))
	}

	if err := s.SetPublicConfig(r.Context(), db, nodeID, iface); err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}

func (s *NodeAPI) configureFreeToUse(r *http.Request) (interface{}, mw.Response) {
	db := mw.Database(r)
	nodeID := mux.Vars(r)["node_id"]

	node, err := s.Get(r.Context(), db, nodeID, false)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	// ensure it is the farmer that does the call
	authorized, merr := isFarmerAuthorized(r, node, db)
	if err != nil {
		return nil, merr
	}

	if !authorized {
		return nil, mw.Forbidden(fmt.Errorf("only the farmer can configured the if the node is free to use"))
	}

	choice := struct {
		FreeToUse bool `json:"free_to_use"`
	}{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&choice); err != nil {
		return nil, mw.BadRequest(err)
	}

	if err := s.updateFreeToUse(r.Context(), db, node.NodeId, choice.FreeToUse); err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Ok()
}

func (s *NodeAPI) registerPorts(r *http.Request) (interface{}, mw.Response) {

	defer r.Body.Close()

	nodeID := mux.Vars(r)["node_id"]
	hNodeID := httpsig.KeyIDFromContext(r.Context())
	if nodeID != hNodeID {
		return nil, mw.Forbidden(fmt.Errorf("trying to register ports for nodeID %s while you are %s", nodeID, hNodeID))
	}

	input := struct {
		Ports []uint `json:"ports"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, mw.BadRequest(err)
	}

	log.Debug().Uints("ports", input.Ports).Msg("wireguard ports received")

	db := mw.Database(r)
	if err := s.SetWGPorts(r.Context(), db, nodeID, input.Ports); err != nil {
		return nil, mw.NotFound(err)
	}

	return nil, nil
}

func (s *NodeAPI) updateUptimeHandler(r *http.Request) (interface{}, mw.Response) {
	defer r.Body.Close()

	nodeID := mux.Vars(r)["node_id"]
	hNodeID := httpsig.KeyIDFromContext(r.Context())
	if nodeID != hNodeID {
		return nil, mw.Forbidden(fmt.Errorf("trying to register uptime for nodeID %s while you are %s", nodeID, hNodeID))
	}

	input := struct {
		Uptime uint64 `json:"uptime"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, mw.BadRequest(err)
	}

	db := mw.Database(r)
	log.Debug().Str("node", nodeID).Uint64("uptime", input.Uptime).Msg("node uptime received")

	if err := s.updateUptime(r.Context(), db, nodeID, int64(input.Uptime)); err != nil {
		return nil, mw.NotFound(err)
	}

	return nil, nil
}

func (s *NodeAPI) updateReservedResources(r *http.Request) (interface{}, mw.Response) {
	defer r.Body.Close()

	nodeID := mux.Vars(r)["node_id"]
	hNodeID := httpsig.KeyIDFromContext(r.Context())
	if nodeID != hNodeID {
		return nil, mw.Forbidden(fmt.Errorf("trying to update reserved capacity for nodeID %s while you are %s", nodeID, hNodeID))
	}

	input := struct {
		generated.ResourceAmount
		generated.WorkloadAmount
	}{}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, mw.BadRequest(err)
	}

	db := mw.Database(r)
	if err := s.updateReservedCapacity(r.Context(), db, nodeID, input.ResourceAmount); err != nil {
		return nil, mw.NotFound(err)
	}
	if err := s.updateWorkloadsAmount(r.Context(), db, nodeID, input.WorkloadAmount); err != nil {
		return nil, mw.NotFound(err)
	}

	return nil, nil
}

func farmOwner(ctx context.Context, farmID int64, db *mongo.Database) (int64, mw.Response) {
	ff := types.FarmFilter{}
	ff = ff.WithID(schema.ID(farmID))

	farm, err := ff.Get(ctx, db)
	if err != nil {
		return 0, mw.Error(err) //TODO
	}

	return farm.ThreebotId, nil
}

// isFarmerAuthorized ensure it is the farmer authenticated in request r is owning the node
func isFarmerAuthorized(r *http.Request, node directory.Node, db *mongo.Database) (bool, mw.Response) {
	actualFarmerID, merr := farmOwner(r.Context(), node.FarmId, db)
	if merr != nil {
		return false, merr
	}

	sfarmerID := httpsig.KeyIDFromContext(r.Context())
	requestFarmerID, err := strconv.ParseInt(sfarmerID, 10, 64)
	if err != nil {
		return false, mw.BadRequest(err)
	}
	log.Debug().Int64("actualFarmerID", actualFarmerID).Int64("requestFarmID", requestFarmerID).Send()
	return (requestFarmerID == actualFarmerID), nil
}
