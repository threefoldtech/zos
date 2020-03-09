package types

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models"
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	reservationCollection = "reservation"
)

const (
	Create  = generated.TfgridWorkloadsReservation1NextActionCreate
	Sign    = generated.TfgridWorkloadsReservation1NextActionSign
	Pay     = generated.TfgridWorkloadsReservation1NextActionPay
	Deploy  = generated.TfgridWorkloadsReservation1NextActionDeploy
	Delete  = generated.TfgridWorkloadsReservation1NextActionDelete
	Invalid = generated.TfgridWorkloadsReservation1NextActionInvalid
	Deleted = generated.TfgridWorkloadsReservation1NextActionDeleted
)

// ReservationFilter type
type ReservationFilter bson.D

// WithID filter reservation with ID
func (f ReservationFilter) WithID(id schema.ID) ReservationFilter {
	return append(f, bson.E{Key: "_id", Value: id})
}

// WithIdGE return find reservations with
func (f ReservationFilter) WithIdGE(id schema.ID) ReservationFilter {
	return append(f, bson.E{
		Key: "_id", Value: bson.M{"$gte": id},
	})
}

// WithNextAction filter reservations with next action
func (f ReservationFilter) WithNextAction(action generated.TfgridWorkloadsReservation1NextActionEnum) ReservationFilter {
	return append(f, bson.E{
		Key: "next_action", Value: action,
	})
}

// WithNodeID searsch reservations with NodeID
func (f ReservationFilter) WithNodeID(id string) ReservationFilter {
	//data_reservation.{containers, volumes, zdbs, networks, kubernetes}.node_id
	// we need to search ALL types for any reservation that has the node ID
	or := []bson.M{}
	for _, typ := range []string{"containers", "volumes", "zdbs", "kubernetes"} {
		key := fmt.Sprintf("data_reservation.%s.node_id", typ)
		or = append(or, bson.M{key: id})
	}

	// network workload is special because node id is set on the network_resources.
	or = append(or, bson.M{"data_reservation.networks.network_resources.node_id": id})

	// we find any reservation that has this node ID set.
	return append(f, bson.E{Key: "$or", Value: or})
}

// Or returns filter that reads as (f or o)
func (f ReservationFilter) Or(o ReservationFilter) ReservationFilter {
	return ReservationFilter{
		bson.E{
			Key:   "$or",
			Value: bson.A{f, o},
		},
	}
}

// Get gets single reservation that matches the filter
func (f ReservationFilter) Get(ctx context.Context, db *mongo.Database) (reservation Reservation, err error) {
	if f == nil {
		f = ReservationFilter{}
	}

	result := db.Collection(reservationCollection).FindOne(ctx, f)
	if err = result.Err(); err != nil {
		return
	}

	err = result.Decode(&reservation)
	return
}

// Find all users that matches filter
func (f ReservationFilter) Find(ctx context.Context, db *mongo.Database, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	if f == nil {
		f = ReservationFilter{}
	}
	return db.Collection(reservationCollection).Find(ctx, f, opts...)
}

// Reservation is a wrapper around generated type
type Reservation generated.TfgridWorkloadsReservation1

// Validate that the reservation is valid
func (r *Reservation) validate() error {
	if r.CustomerTid == 0 {
		return fmt.Errorf("customer_tid is required")
	}

	if len(r.CustomerSignature) == 0 {
		return fmt.Errorf("customer_signature is required")
	}

	var data generated.TfgridWorkloadsReservationData1

	if err := json.Unmarshal([]byte(r.Json), &data); err != nil {
		return errors.Wrap(err, "invalid json data on reservation")
	}

	if !reflect.DeepEqual(r.DataReservation, data) {
		return fmt.Errorf("json data does not match the reservation data")
	}

	ids := make(map[int64]struct{})

	// yes, it's ugly. live with it.
	for _, w := range r.DataReservation.Containers {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	for _, w := range r.DataReservation.Networks {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	for _, w := range r.DataReservation.Zdbs {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	for _, w := range r.DataReservation.Volumes {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	for _, w := range r.DataReservation.Kubernetes {
		if _, ok := ids[w.WorkloadId]; ok {
			return fmt.Errorf("conflicting workload ID '%d'", w.WorkloadId)
		}
		ids[w.WorkloadId] = struct{}{}
	}

	return nil
}

// Verify signature against Reserveration.JSON
// pk is the public key used as verification key in hex encoded format
// the signature is the signature to verify (in raw binary format)
func (r *Reservation) Verify(pk string, sig []byte) error {
	key, err := crypto.KeyFromHex(pk)
	if err != nil {
		return errors.Wrap(err, "invalid verification key")
	}

	return crypto.Verify(key, []byte(r.Json), sig)
}

// Expired checks if this reservation has expired
func (r *Reservation) Expired() bool {
	if time.Until(r.DataReservation.ExpirationReservation.Time) <= 0 {
		return true
	}

	return false
}

// IsAny checks if the reservation status is any of the given status
func (r *Reservation) IsAny(status ...generated.TfgridWorkloadsReservation1NextActionEnum) bool {
	for _, s := range status {
		if r.NextAction == s {
			return true
		}
	}

	return false
}

//ResultOf return result of a workload ID
func (r *Reservation) ResultOf(id string) *Result {
	for _, result := range r.Results {
		if result.WorkloadId == id {
			r := Result(result)
			return &r
		}
	}

	return nil
}

// AllDeleted checks of all workloads has been marked
func (r *Reservation) AllDeleted() bool {
	// check if all workloads have been deleted.
	for _, wl := range r.Workloads("") {
		result := r.ResultOf(wl.WorkloadId)
		if result == nil ||
			result.State != generated.TfgridWorkloadsReservationResult1StateDeleted {
			return false
		}
	}

	return true
}

// Workloads returns all reservation workloads (filter by nodeID)
// if nodeID is empty, return all workloads
func (r *Reservation) Workloads(nodeID string) []Workload {
	data := &r.DataReservation
	var workloads []Workload
	for _, wl := range data.Containers {
		if len(nodeID) > 0 && wl.NodeId != nodeID {
			continue
		}
		workload := Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeContainer,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: wl.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, wl := range data.Volumes {
		if len(nodeID) > 0 && wl.NodeId != nodeID {
			continue
		}
		workload := Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeVolume,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: wl.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, wl := range data.Zdbs {
		if len(nodeID) > 0 && wl.NodeId != nodeID {
			continue
		}
		workload := Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeZdb,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: wl.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, wl := range data.Kubernetes {
		if len(nodeID) > 0 && wl.NodeId != nodeID {
			continue
		}
		workload := Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeKubernetes,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: wl.NodeId,
		}

		workloads = append(workloads, workload)
	}

	for _, wl := range data.Networks {
		found := false
		if len(nodeID) > 0 {
			for _, nr := range wl.NetworkResources {
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
					  but for different  node IDs. this means that multiple node will report
					  result with the same Global Workload ID. but we only store the
					  last stored result. Hence we lose the status for other network resources
					  deployments.
					  I think the network workload needs to have different workload ids per
					  network resource.
					  Thoughts?
		*/
		workload := Workload{
			TfgridWorkloadsReservationWorkload1: generated.TfgridWorkloadsReservationWorkload1{
				WorkloadId: fmt.Sprintf("%d-%d", r.ID, wl.WorkloadId),
				User:       fmt.Sprint(r.CustomerTid),
				Type:       generated.TfgridWorkloadsReservationWorkload1TypeNetwork,
				Content:    wl,
				Created:    r.Epoch,
				Duration:   int64(data.ExpirationReservation.Sub(r.Epoch.Time).Seconds()),
				ToDelete:   r.NextAction == Delete || r.NextAction == Deleted,
			},
			NodeID: nodeID,
		}

		workloads = append(workloads, workload)
	}

	return workloads
}

// ReservationCreate save new reservation to database.
// NOTE: use reservations only that are returned from calling Pipeline.Next()
// no validation is done here, this is just a CRUD operation
func ReservationCreate(ctx context.Context, db *mongo.Database, r Reservation) (schema.ID, error) {
	id := models.MustID(ctx, db, reservationCollection)
	r.ID = id

	_, err := db.Collection(reservationCollection).InsertOne(ctx, r)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// ReservationSetNextAction update the reservation next action in db
func ReservationSetNextAction(ctx context.Context, db *mongo.Database, id schema.ID, action generated.TfgridWorkloadsReservation1NextActionEnum) error {
	var filter ReservationFilter
	filter = filter.WithID(id)

	col := db.Collection(reservationCollection)
	result, err := col.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"next_action": action,
		},
	})

	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("no reservation found")
	}

	return nil
}

//ReservationPushSignature push signature to reservation
func ReservationPushSignature(ctx context.Context, db *mongo.Database, id schema.ID, signature generated.TfgridWorkloadsReservationSigningSignature1) error {

	var filter ReservationFilter
	filter = filter.WithID(id)
	col := db.Collection(reservationCollection)
	// NOTE: this should be a transaction not a bulk write
	// but i had so many issues with transaction, and i couldn't
	// get it to work. so I used bulk write in place instead
	// until we figure this issue out.
	// Note, the reason we don't just use addToSet is the signature
	// object always have the current 'time' which means it's a different
	// value than the one in the document even if it has same user id.
	_, err := col.BulkWrite(ctx, []mongo.WriteModel{
		mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(
			bson.M{
				"$pull": bson.M{
					"signatures_provision": bson.M{"tid": signature.Tid},
				},
			}),
		mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(
			bson.M{
				"$addToSet": bson.M{
					"signatures_provision": signature,
				},
			}),
	}, options.BulkWrite().SetOrdered(true))

	return err
}

// Workload is a wrapper around generated TfgridWorkloadsReservationWorkload1 type
type Workload struct {
	generated.TfgridWorkloadsReservationWorkload1
	NodeID string `json:"-" bson:"-"`
}

// Result is a wrapper around TfgridWorkloadsReservationResult1 type
type Result generated.TfgridWorkloadsReservationResult1

func (r *Result) encode() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := buf.WriteByte(byte(r.State)); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(r.Message); err != nil {
		return nil, err
	}
	if _, err := buf.Write(r.DataJson); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Verify that the signature matches the result data
func (r *Result) Verify(pk string) error {
	sig, err := hex.DecodeString(r.Signature)
	if err != nil {
		return errors.Wrap(err, "invalid signature expecting hex encoded")
	}

	key, err := crypto.KeyFromID(pkg.StrIdentifier(pk))
	if err != nil {
		return errors.Wrap(err, "invalid verification key")
	}

	bytes, err := r.encode()
	if err != nil {
		return err
	}

	return crypto.Verify(key, bytes, sig)
}

// PushResult pushes result to a reservation result array.
// NOTE: this is just a crud operation, no validation is done here
func PushResult(ctx context.Context, db *mongo.Database, id schema.ID, result Result) error {
	col := db.Collection(reservationCollection)
	var filter ReservationFilter
	filter = filter.WithID(id)

	// we don't care if we couldn't delete old result.
	// in case it never existed, or the array is nil.
	col.UpdateOne(ctx, filter, bson.M{
		"$pull": bson.M{
			"results": bson.M{
				"workload_id": result.WorkloadId,
			},
		},
	})

	_, err := col.UpdateOne(ctx, filter, bson.D{
		{
			Key: "$push",
			Value: bson.M{
				"results": result,
			},
		},
	})

	return err
}
