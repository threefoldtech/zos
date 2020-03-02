package workloads

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

	// we can't start with any of the signatures
	reservation.SignaturesProvision = nil
	reservation.SignaturesDelete = nil
	reservation.SignaturesFarmer = nil

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

func (a *API) workloadsFromReserveration(nodeID string, reservation *types.Reservation) []types.Workload {
	data := &reservation.DataReservation
	var workloads []types.Workload
	for _, r := range data.Containers {
		if r.NodeId != nodeID {
			continue
		}
		workload := types.Workload{
			WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
			User:       fmt.Sprint(reservation.CustomerTid),
			Type:       generated.TfgridWorkloadsReservationWorkload1TypeContainer,
			Content:    r,
			Created:    reservation.Epoch,
			Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
			ToDelete:   reservation.NextAction == types.Delete,
		}

		workloads = append(workloads, workload)
	}

	for _, r := range data.Volumes {
		if r.NodeId != nodeID {
			continue
		}
		workload := types.Workload{
			WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
			User:       fmt.Sprint(reservation.CustomerTid),
			Type:       generated.TfgridWorkloadsReservationWorkload1TypeVolume,
			Content:    r,
			Created:    reservation.Epoch,
			Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
			ToDelete:   reservation.NextAction == types.Delete,
		}

		workloads = append(workloads, workload)
	}

	for _, r := range data.Zdbs {
		if r.NodeId != nodeID {
			continue
		}
		workload := types.Workload{
			WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
			User:       fmt.Sprint(reservation.CustomerTid),
			Type:       generated.TfgridWorkloadsReservationWorkload1TypeZdb,
			Content:    r,
			Created:    reservation.Epoch,
			Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
			ToDelete:   reservation.NextAction == types.Delete,
		}

		workloads = append(workloads, workload)
	}

	for _, r := range data.Kubernetes {
		if r.NodeId != nodeID {
			continue
		}
		workload := types.Workload{
			WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
			User:       fmt.Sprint(reservation.CustomerTid),
			Type:       generated.TfgridWorkloadsReservationWorkload1TypeKubernetes,
			Content:    r,
			Created:    reservation.Epoch,
			Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
			ToDelete:   reservation.NextAction == types.Delete,
		}

		workloads = append(workloads, workload)
	}

	for _, r := range data.Networks {
		found := false
		for _, nr := range r.NetworkResources {
			if nr.NodeId == nodeID {
				found = true
				break
			}
		}

		if !found {
			continue
		}

		workload := types.Workload{
			WorkloadId: fmt.Sprintf("%d-%d", reservation.ID, r.WorkloadId),
			User:       fmt.Sprint(reservation.CustomerTid),
			Type:       generated.TfgridWorkloadsReservationWorkload1TypeNetwork,
			Content:    r,
			Created:    reservation.Epoch,
			Duration:   int64(data.ExpirationReservation.Sub(reservation.Epoch.Time).Seconds()),
			ToDelete:   reservation.NextAction == types.Delete,
		}

		workloads = append(workloads, workload)
	}

	return workloads
}

func (a *API) workloads(r *http.Request) (interface{}, mw.Response) {
	const (
		maxPageSize = 100
	)

	var (
		nodeID = mux.Vars(r)["node_id"]
	)

	from, err := a.parseID(r.FormValue("from"))
	if err != nil {
		return nil, mw.BadRequest(err)
	}

	var filter types.ReservationFilter
	filter = filter.WithNodeID(nodeID).WithIdGE(from)
	log.Debug().Msgf("filter: %+v", filter)
	db := mw.Database(r)
	cur, err := filter.Find(r.Context(), db)
	if err != nil {
		return nil, mw.Error(err)
	}

	defer cur.Close(r.Context())

	workloads := []types.Workload{}
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

	a.updateMany(db, needUpdate)

	return workloads, nil
}
