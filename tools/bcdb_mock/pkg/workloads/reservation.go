package workloads

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	phonebook "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/phonebook/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/workloads/types"
	"go.mongodb.org/mongo-driver/mongo"
)

// API struct
type API struct{}

func (a *API) create(r *http.Request) (interface{}, mw.Response) {
	defer r.Body.Close()
	var reservation types.Reservation
	if err := json.NewDecoder(r.Body).Decode(&reservation); err != nil {
		return nil, mw.BadRequest(err)
	}

	if reservation.Expired() {
		return nil, mw.BadRequest(fmt.Errorf("creating for a reservation that expires in the past"))
	}

	// we make sure those arrays are initialized correctly
	// this will make updating the document in place much easier
	// in later stages
	reservation.SignaturesProvision = make([]generated.TfgridWorkloadsReservationSigningSignature1, 0)
	reservation.SignaturesDelete = make([]generated.TfgridWorkloadsReservationSigningSignature1, 0)
	reservation.SignaturesFarmer = make([]generated.TfgridWorkloadsReservationSigningSignature1, 0)
	reservation.Results = make([]generated.TfgridWorkloadsReservationResult1, 0)

	pipeline, err := types.NewPipeline(reservation)
	if err != nil {
		// if failed to create pipeline, then
		// this reservation has failed initial validation
		return nil, mw.BadRequest(err)
	}

	// we need to save it anyway, so no need to check
	// if reservation status has changed
	reservation, _ = pipeline.Next()

	if reservation.IsAny(types.Invalid, types.Delete) {
		return nil, mw.BadRequest(fmt.Errorf("invalid request wrong status '%s'", reservation.NextAction.String()))
	}

	db := mw.Database(r)
	var filter phonebook.UserFilter
	filter = filter.WithID(schema.ID(reservation.CustomerTid))
	user, err := filter.Get(r.Context(), db)
	if err != nil {
		return nil, mw.BadRequest(errors.Wrapf(err, "cannot find user with id '%d'", reservation.CustomerTid))
	}

	signature, err := hex.DecodeString(reservation.CustomerSignature)
	if err := reservation.Verify(user.Pubkey, signature); err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "failed to verify customer signature"))
	}

	reservation.Epoch = schema.Date{Time: time.Now()}

	id, err := types.ReservationCreate(r.Context(), db, reservation)
	if err != nil {
		return nil, mw.Error(err)
	}

	return id, mw.Created()
}

func (a *API) parseID(id string) (schema.ID, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "invalid id format")
	}

	return schema.ID(v), nil
}

func (a *API) get(r *http.Request) (interface{}, mw.Response) {
	id, err := a.parseID(mux.Vars(r)["res_id"])
	if err != nil {
		return nil, mw.BadRequest(fmt.Errorf("invalid reservation id"))
	}

	var filter types.ReservationFilter
	filter = filter.WithID(id)

	db := mw.Database(r)
	reservation, err := filter.Get(r.Context(), db)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	return reservation, nil
}

func (a *API) updateMany(db *mongo.Database, rs []types.Reservation) {
	if len(rs) == 0 {
		return
	}
}

func (a *API) list(r *http.Request) (interface{}, mw.Response) {
	var filter types.ReservationFilter

	db := mw.Database(r)
	cur, err := filter.Find(r.Context(), db, models.PageFromRequest(r))
	if err != nil {
		return nil, mw.Error(err)
	}

	defer cur.Close(r.Context())

	reservations := []types.Reservation{}

	var needUpdate []types.Reservation
	for cur.Next(r.Context()) {
		var reservation types.Reservation
		if err := cur.Decode(&reservation); err != nil {
			return nil, mw.Error(err)
		}

		pl, err := types.NewPipeline(reservation)
		if err != nil {
			log.Error().Err(err).Int64("id", int64(reservation.ID)).Msg("failed to process reservation")
			continue
		}

		reservation, update := pl.Next()
		if update {
			needUpdate = append(needUpdate, reservation)
		}

		reservations = append(reservations, reservation)
	}

	a.updateMany(db, needUpdate)

	return reservations, nil
}

func (a *API) markDelete(r *http.Request) (interface{}, mw.Response) {
	// WARNING: #TODO
	// This method does not validate the signature of the caller
	// because there is no payload in a delete call.
	// may be a simple body that has "reservation id" and "signature"
	// can be used, we use the reservation id to avoid using the same
	// request body to delete other reservations

	// HTTP Delete should not have a body though, so may be this should be
	// changed to a PUT operation.

	id, err := a.parseID(mux.Vars(r)["res_id"])
	if err != nil {
		return nil, mw.Error(err)
	}

	var filter types.ReservationFilter
	filter = filter.WithID(id)
	db := mw.Database(r)
	reservation, err := filter.Get(r.Context(), db)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	if reservation.NextAction == generated.TfgridWorkloadsReservation1NextActionDeleted ||
		reservation.NextAction == generated.TfgridWorkloadsReservation1NextActionDelete {
		return nil, mw.BadRequest(fmt.Errorf("resource already deleted"))
	}

	if err = types.ReservationSetNextAction(r.Context(), db, id, generated.TfgridWorkloadsReservation1NextActionDelete); err != nil {
		return nil, mw.Error(err)
	}

	return nil, nil
}

func (a *API) workloadsFromReserveration(nodeID string, reservation *types.Reservation) []types.Workload {
	data := &reservation.DataReservation
	var workloads []types.Workload
	for _, r := range data.Containers {
		if len(nodeID) > 0 && r.NodeId != nodeID {
			continue
		}
		workload := types.Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
				User:       fmt.Sprint(reservation.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeContainer,
				Content:    r,
				Created:    reservation.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
				ToDelete:   reservation.NextAction == types.Delete,
			},
			NodeID: r.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, r := range data.Volumes {
		if len(nodeID) > 0 && r.NodeId != nodeID {
			continue
		}
		workload := types.Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
				User:       fmt.Sprint(reservation.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeVolume,
				Content:    r,
				Created:    reservation.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
				ToDelete:   reservation.NextAction == types.Delete,
			},
			NodeID: r.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, r := range data.Zdbs {
		if len(nodeID) > 0 && r.NodeId != nodeID {
			continue
		}
		workload := types.Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
				User:       fmt.Sprint(reservation.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeZdb,
				Content:    r,
				Created:    reservation.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
				ToDelete:   reservation.NextAction == types.Delete,
			},
			NodeID: r.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, r := range data.Kubernetes {
		if len(nodeID) > 0 && r.NodeId != nodeID {
			continue
		}
		workload := types.Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
				User:       fmt.Sprint(reservation.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeKubernetes,
				Content:    r,
				Created:    reservation.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
				ToDelete:   reservation.NextAction == types.Delete,
			},
			NodeID: r.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, r := range data.Networks {
		found := false
		if len(nodeID) > 0 {
			for _, nr := range r.NetworkResources {
				if nr.NodeId == nodeID {
					found = true
					break
				}
			}
		} else {
			// if node id is not set, we list all workloads
			found = true
		}

		if !found {
			continue
		}

		/*
			QUESTION: we will have identical workloads (one per each network resource)
					  but for different IDs. this means that multiple node will report
					  result with the same Gloabal Workload ID. but we only store the
					  last stored result. Hence we lose the status for other network resources
					  deployments.
					  I think the network workload needs to have different workload ids per
					  network resource.
					  Thoughts?
		*/
		workload := types.Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
				User:       fmt.Sprint(reservation.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeNetwork,
				Content:    r,
				Created:    reservation.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
				ToDelete:   reservation.NextAction == types.Delete,
			},
			NodeID: nodeID,
		}

		workloads = append(workloads, workload)
	}

	return workloads
}

func (a *API) workloads(r *http.Request) (interface{}, mw.Response) {
	const (
		maxPageSize = 50
	)

	var (
		nodeID = mux.Vars(r)["node_id"]
	)

	from, err := a.parseID(r.FormValue("from"))
	if err != nil {
		return nil, mw.BadRequest(err)
	}

	find := func(ctx context.Context, db *mongo.Database, filter types.ReservationFilter) ([]types.Workload, error) {
		cur, err := filter.Find(r.Context(), db)
		if err != nil {
			return nil, err
		}

		defer cur.Close(r.Context())

		workloads := []types.Workload{}

		for cur.Next(r.Context()) {
			var reservation types.Reservation
			if err := cur.Decode(&reservation); err != nil {
				return nil, err
			}
			pl, err := types.NewPipeline(reservation)
			if err != nil {
				log.Error().Err(err).Int64("id", int64(reservation.ID)).Msg("failed to process reservation")
				continue
			}

			reservation, _ = pl.Next()

			// only reservations that is in right
			if reservation.IsAny(types.Deploy, types.Delete) {
				workloads = append(
					workloads,
					a.workloadsFromReserveration(nodeID, &reservation)...,
				)
			}

			if len(workloads) >= maxPageSize {
				break
			}
		}

		return workloads, nil
	}

	db := mw.Database(r)

	// first we find ALL reservations that has the Delete flag set
	var filter types.ReservationFilter
	filter = filter.WithNodeID(nodeID).WithNextAction(generated.TfgridWorkloadsReservation1NextActionDelete)

	workloads, err := find(r.Context(), db, filter)
	if err != nil {
		return nil, mw.Error(errors.Wrap(err, "failed to list reservations to delete"))
	}

	filter = types.ReservationFilter{}
	filter = filter.WithNodeID(nodeID).WithIdGE(from)

	toCreate, err := find(r.Context(), db, filter)
	if err != nil {
		return nil, mw.Error(errors.Wrap(err, "failed to list new reservations"))
	}
	// TODO: unify the query so we don't have duplicates
	workloads = append(workloads, toCreate...)
	return workloads, nil
}

func (a *API) workloadGet(r *http.Request) (interface{}, mw.Response) {
	gwid := mux.Vars(r)["gwid"]

	rid, err := a.parseID(strings.Split(gwid, "-")[0])
	if err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "invalid reservation id part"))
	}

	var filter types.ReservationFilter
	filter = filter.WithID(rid)

	db := mw.Database(r)
	reservation, err := filter.Get(r.Context(), db)
	if err != nil {
		return nil, mw.NotFound(err)
	}
	// we use an empty node-id in listing to return all workloads in this reservation
	workloads := a.workloadsFromReserveration("", &reservation)

	var workload *types.Workload
	for _, wl := range workloads {
		if wl.WorkloadId == gwid {
			workload = &wl
			break
		}
	}

	if workload == nil {
		return nil, mw.NotFound(fmt.Errorf("workload not found"))
	}
	var result struct {
		types.Workload
		Result *types.Result `json:"result"`
	}
	result.Workload = *workload
	for _, rs := range reservation.Results {
		if rs.WorkloadId == workload.WorkloadId {
			t := types.Result(rs)
			result.Result = &t
			break
		}
	}

	return result, nil
}

func (a *API) workloadPutResult(r *http.Request) (interface{}, mw.Response) {
	defer r.Body.Close()

	nodeID := mux.Vars(r)["node_id"]
	gwid := mux.Vars(r)["gwid"]

	rid, err := a.parseID(strings.Split(gwid, "-")[0])
	if err != nil {
		return nil, mw.BadRequest(errors.Wrap(err, "invalid reservation id part"))
	}

	var result types.Result
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		return nil, mw.BadRequest(err)
	}

	var filter types.ReservationFilter
	filter = filter.WithID(rid)

	db := mw.Database(r)
	reservation, err := filter.Get(r.Context(), db)
	if err != nil {
		return nil, mw.NotFound(err)
	}
	// we use an empty node-id in listing to return all workloads in this reservation
	workloads := a.workloadsFromReserveration(nodeID, &reservation)
	var workload *types.Workload
	for _, wl := range workloads {
		if wl.WorkloadId == gwid {
			workload = &wl
			break
		}
	}

	if workload == nil {
		return nil, mw.NotFound(errors.New("workload not found"))
	}

	result.WorkloadId = gwid
	result.Epoch = schema.Date{Time: time.Now()}

	if err := result.Verify(nodeID); err != nil {
		return nil, mw.UnAuthorized(errors.Wrap(err, "invalid result signature"))
	}

	if err := types.PushResult(r.Context(), db, rid, result); err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}
