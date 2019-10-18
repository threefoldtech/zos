package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/network/types"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
)

func (s *nodeStore) registerNode(w http.ResponseWriter, r *http.Request) {
	log.Println("node register request received")

	defer r.Body.Close()

	n := directory.TfgridNode2{}
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	if err := s.Add(n); err != nil {
		httpError(w, err, http.StatusInternalServerError)
		return
	}
	log.Printf("node registered: %+v\n", n)

	w.WriteHeader(http.StatusCreated)
}

func (s *nodeStore) nodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := mux.Vars(r)["node_id"]
	node, err := s.Get(nodeID)
	if err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&node); err != nil {
		log.Printf("error writing node: %v", err)
	}
}

func (s *nodeStore) listNodes(w http.ResponseWriter, r *http.Request) {
	nodes := s.List()
	farm := r.URL.Query().Get("farm")

	for i, node := range nodes {
		if node == nil {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}

		if farm != "" && node.FarmID != farm {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(nodes)
}

func (s *nodeStore) cockpitListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := s.List()
	farm := r.URL.Query().Get("farm")

	for i, node := range nodes {
		if node == nil {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}

		if farm != "" && node.FarmID != farm {
			nodes = append(nodes[:i], nodes[i+1:]...)
			continue
		}
	}

	x := struct {
		Node []*directory.TfgridNode2 `json:"nodes"`
	}{nodes}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(x)
}

func (s *nodeStore) registerCapacity(w http.ResponseWriter, r *http.Request) {
	x := struct {
		Capacity directory.TfgridNodeResourceAmount1 `json:"capacity,omitempty"`
		DMI      *dmi.DMI                            `json:"dmi,omitempty"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	nodeID := mux.Vars(r)["node_id"]
	if err := s.updateTotalCapacity(nodeID, x.Capacity); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (s *nodeStore) registerIfaces(w http.ResponseWriter, r *http.Request) {
	log.Println("network interfaces register request received")

	defer r.Body.Close()

	input := []*types.IfaceInfo{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf(err.Error())
		httpError(w, err, http.StatusBadRequest)
		return
	}

	log.Println("network interfaces received", input)
	ifaces := make([]directory.TfgridNodeIface1, len(input))
	for i, iface := range input {
		ifaces[i].Gateway = iface.Gateway
		ifaces[i].Name = iface.Name
		for _, r := range iface.Addrs {
			ifaces[i].Addrs = append(ifaces[i].Addrs, r.ToSchema())
		}
	}

	nodeID := mux.Vars(r)["node_id"]
	if err := s.SetInterfaces(nodeID, ifaces); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *nodeStore) configurePublic(w http.ResponseWriter, r *http.Request) {
	iface := types.PubIface{}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&iface); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	cfg := directory.TfgridNodePublicIface1{
		Gw4:     iface.GW4,
		Gw6:     iface.GW6,
		Master:  iface.Master,
		Type:    directory.TfgridNodePublicIface1TypeMacvlan,
		Version: int64(iface.Version),
	}

	nodeID := mux.Vars(r)["node_id"]
	if err := s.SetPublicConfig(nodeID, cfg); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *nodeStore) registerPorts(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()

	input := struct {
		Ports []uint `json:"ports"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httpError(w, err, http.StatusBadRequest)
		return
	}

	fmt.Println("wireguard ports received", input.Ports)

	nodeID := mux.Vars(r)["node_id"]
	if err := s.SetWGPorts(nodeID, input.Ports); err != nil {
		httpError(w, err, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// func registerFarm(w http.ResponseWriter, r *http.Request) {
// 	log.Println("farm register request received")

// 	defer r.Body.Close()

// 	info := farmInfo{}
// 	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	farmStore[info.ID] = &info
// 	w.WriteHeader(http.StatusCreated)
// }

// func listFarm(w http.ResponseWriter, r *http.Request) {
// 	var farms = make([]*farmInfo, 0, len(farmStore))
// 	for _, info := range farmStore {
// 		farms = append(farms, info)
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	_ = json.NewEncoder(w).Encode(farms)
// }

// func getFarm(w http.ResponseWriter, r *http.Request) {
// 	farmID := mux.Vars(r)["farm_id"]
// 	farm, ok := farmStore[farmID]
// 	if !ok {
// 		http.Error(w, fmt.Sprintf("farm %s not found", farmID), http.StatusNotFound)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	_ = json.NewEncoder(w).Encode(farm)
// }
