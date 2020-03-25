package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
)

type httpWorkloads struct {
	*httpClient
}

func (w *httpWorkloads) Create(reservation workloads.TfgridWorkloadsReservation1) (id schema.ID, err error) {
	err = w.post(w.url("reservations"), reservation, &id, http.StatusCreated)
	return
}

func (w *httpWorkloads) List(page *Pager) (reservation []workloads.TfgridWorkloadsReservation1, err error) {
	query := url.Values{}
	page.apply(query)

	err = w.get(w.url("reservations"), query, &reservation, http.StatusOK)
	return
}

func (w *httpWorkloads) Get(id schema.ID) (reservation workloads.TfgridWorkloadsReservation1, err error) {
	err = w.get(w.url("reservations", fmt.Sprint(id)), nil, &reservation, http.StatusOK)
	return
}

func (w *httpWorkloads) SignProvision(id schema.ID, user schema.ID, signature string) error {
	return w.post(
		w.url("reservations", fmt.Sprint(id), "sign", "provision"),
		workloads.TfgridWorkloadsReservationSigningSignature1{
			Tid:       int64(user),
			Signature: signature,
		},
		nil,
		http.StatusCreated,
	)
}

func (w *httpWorkloads) SignDelete(id schema.ID, user schema.ID, signature string) error {
	return w.post(
		w.url("reservations", fmt.Sprint(id), "sign", "delete"),
		workloads.TfgridWorkloadsReservationSigningSignature1{
			Tid:       int64(user),
			Signature: signature,
		},
		nil,
		http.StatusCreated,
	)
}

type intermediateWL struct {
	workloads.TfgridWorkloadsReservationWorkload1
	Content json.RawMessage `json:"content"`
}

func (wl *intermediateWL) Workload() (result workloads.TfgridWorkloadsReservationWorkload1, err error) {
	result = wl.TfgridWorkloadsReservationWorkload1
	switch wl.Type {
	case workloads.TfgridWorkloadsReservationWorkload1TypeContainer:
		var o workloads.TfgridWorkloadsReservationContainer1
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	case workloads.TfgridWorkloadsReservationWorkload1TypeKubernetes:
		var o workloads.TfgridWorkloadsReservationK8S1
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	case workloads.TfgridWorkloadsReservationWorkload1TypeNetwork:
		var o workloads.TfgridWorkloadsReservationNetwork1
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	case workloads.TfgridWorkloadsReservationWorkload1TypeVolume:
		var o workloads.TfgridWorkloadsReservationVolume1
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	case workloads.TfgridWorkloadsReservationWorkload1TypeZdb:
		var o workloads.TfgridWorkloadsReservationZdb1
		if err := json.Unmarshal(wl.Content, &o); err != nil {
			return result, err
		}
		result.Content = o
	default:
		return result, fmt.Errorf("unknown workload type")
	}

	return
}

func (w *httpWorkloads) Workloads(nodeID string, from uint64) ([]workloads.TfgridWorkloadsReservationWorkload1, error) {
	query := url.Values{}
	query.Set("from", fmt.Sprint(from))

	var list []intermediateWL

	err := w.get(
		w.url("reservations", "workloads", nodeID),
		query,
		&list,
		http.StatusOK,
	)
	if err != nil {
		return nil, err
	}
	results := make([]workloads.TfgridWorkloadsReservationWorkload1, 0, len(list))
	for _, i := range list {
		wl, err := i.Workload()
		if err != nil {
			return nil, err
		}
		results = append(results, wl)
	}
	return results, err
}

func (w *httpWorkloads) WorkloadGet(gwid string) (result workloads.TfgridWorkloadsReservationWorkload1, err error) {
	var output intermediateWL
	err = w.get(w.url("reservations", "workloads", gwid), nil, &output, http.StatusOK)
	if err != nil {
		return
	}

	return output.Workload()
}

func (w *httpWorkloads) WorkloadPutResult(nodeID, gwid string, result workloads.TfgridWorkloadsReservationResult1) error {
	return w.put(w.url("reservations", "workloads", gwid, nodeID), result, nil, http.StatusCreated)
}

func (w *httpWorkloads) WorkloadPutDeleted(nodeID, gwid string) error {
	return w.delete(w.url("reservations", "workloads", gwid, nodeID), nil, nil, http.StatusOK)
}
