package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	"github.com/threefoldtech/zos/tools/bcdb_mock/types/directory"

	"github.com/gorilla/mux"
)

func (s *FarmAPI) registerFarm(r *http.Request) (interface{}, Response) {
	log.Println("farm register request received")

	db := mw.Database(r)
	defer r.Body.Close()

	var info directory.Farm
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		return nil, BadRequest(err)
	}

	if err := info.Validate(); err != nil {
		return nil, BadRequest(err)
	}

	id, err := s.Add(r.Context(), db, info)
	if err != nil {
		return nil, Error(err)
	}

	return struct {
		ID schema.ID `json:"id"`
	}{
		id,
	}, Created()
}

func (s *FarmAPI) listFarm(r *http.Request) (interface{}, Response) {
	db := mw.Database(r)
	farms, err := s.List(r.Context(), db)
	if err != nil {
		return nil, Error(err)
	}

	return farms, nil
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

func (s *FarmAPI) getFarm(r *http.Request) (interface{}, Response) {
	sid := mux.Vars(r)["farm_id"]

	id, err := strconv.ParseInt(sid, 10, 64)
	if err != nil {
		return nil, BadRequest(err)
	}

	db := mw.Database(r)

	farm, err := s.GetByID(r.Context(), db, id)
	if err != nil {
		return nil, NotFound(err)
	}

	return farm, nil
}
