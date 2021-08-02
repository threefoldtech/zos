package yggdrasil

import (
	"encoding/json"
	"net/http"
)

//PeerListFallback is an hardcoded list of public yggdrasil node
// it is used to have some available peer to connect to when we failed to read the online public peer info
var PeerListFallback = PeerList{
	{
		Endpoint: "tls://45.147.198.155:6010",
		Up:       true,
	},
	{
		Endpoint: "tcp://51.15.204.214:12345",
		Up:       true,
	},
	{
		Endpoint: "tcp://45.138.172.192:5001",
		Up:       true,
	},
}

// NodeInfo is the know information about an yggdrasil public node
type NodeInfo struct {
	Endpoint   string `json:"-"`
	Up         bool   `json:"up"`
	BoxPubKey  string `json:"key"`
	LastSeen   int    `json:"last_seen"`
	ProtoMinor int    `json:"proto_minor"`
}

// PeerList is yggdrasil peers list
type PeerList []NodeInfo

// FetchPeerList download the list of public yggdrasil peer from https://publicpeers.neilalexander.dev/publicnodes.json
func FetchPeerList() (PeerList, error) {

	var values map[string]map[string]NodeInfo

	resp, err := http.Get("https://publicpeers.neilalexander.dev/publicnodes.json")
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&values); err != nil {
		return nil, err
	}

	var peers PeerList
	for _, nodes := range values {
		for endpoint, info := range nodes {
			if info.ProtoMinor != 4 {
				continue
			}
			info.Endpoint = endpoint
			peers = append(peers, info)
		}
	}

	return peers, nil
}

// Ups return all the peers that are marked up from the PeerList p
func (p PeerList) Ups() PeerList {
	var r PeerList
	for _, x := range p {
		if x.Up {
			r = append(r, x)
		}

	}
	return r
}
