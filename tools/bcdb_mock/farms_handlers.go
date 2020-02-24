package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	"github.com/threefoldtech/zos/tools/bcdb_mock/types/directory"

	"github.com/gorilla/mux"
)

func (s *FarmAPI) registerFarm(w http.ResponseWriter, r *http.Request) {
	log.Println("farm register request received")

	db := mw.Database(r)
	defer r.Body.Close()

	var info directory.Farm
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := s.Add(r.Context(), db, info)
	if err != nil {
		httpError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		ID schema.ID `json:"id"`
	}{
		id,
	})
}

func (s *FarmAPI) listFarm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	db := mw.Database(r)
	farms, err := s.List(r.Context(), db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(farms)
}

func (s *FarmAPI) cockpitListFarm(w http.ResponseWriter, r *http.Request) {
	// TODO: do we need this ?

	// x := struct {
	// 	Farms []*directory.TfgridFarm1 `json:"farms"`
	// }{s.List()}

	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)
	// _ = json.NewEncoder(w).Encode(x)
}

func (s *FarmAPI) getFarm(w http.ResponseWriter, r *http.Request) {
	sid := mux.Vars(r)["farm_id"]

	id, err := strconv.ParseInt(sid, 10, 64)
	if err != nil {
		httpError(w, errors.Wrap(err, "id should be an integer"), http.StatusBadRequest)
		return
	}

	db := mw.Database(r)

	farm, err := s.GetByID(r.Context(), db, id)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(farm)
}
