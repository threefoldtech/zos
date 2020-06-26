package yggdrasil

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// NodeInfo is the know information about an yggdrasil public node
type NodeInfo struct {
	Endpoint  string
	Up        bool
	Address   string
	BoxPubKey string
	LastSeen  int
}

// PeerList is a list of yggsdrasil peer retrieved from https://publicpeers.neilalexander.dev/publicnodes.json
type PeerList struct {
	peers map[string]map[string]NodeInfo
}

// FetchPeerList download the list of public yggdrasil peer from https://publicpeers.neilalexander.dev/publicnodes.json
func FetchPeerList() (PeerList, error) {
	pl := PeerList{}
	fallback := PeerList{
		peers: map[string]map[string]NodeInfo{
			"fallback.md": map[string]NodeInfo{
				"tls://45.147.198.155:6010": {
					Endpoint: "tls://45.147.198.155:6010",
					Up:       true,
				},
				"tcp://85.17.15.221:35239": {
					Endpoint: "tcp://85.17.15.221:35239",
					Up:       true,
				},
				"tcp://51.255.223.60:64982": {
					Endpoint: "tcp://51.255.223.60:64982",
					Up:       true,
				},
			},
		},
	}

	resp, err := http.Get("https://publicpeers.neilalexander.dev/publicnodes.json")
	if err != nil {
		log.Warn().Err(err).Msg("could not fetch public peerlist, using backup")
		return fallback, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&pl.peers); err != nil {
		log.Warn().Err(err).Msg("could not fetch public peerlist, using backup")
		return fallback, nil
	}

	for country := range pl.peers {
		for endpoint := range pl.peers[country] {
			info := pl.peers[country][endpoint]
			info.Endpoint = endpoint
			pl.peers[country][endpoint] = info
		}
	}

	return pl, nil
}

// Peers return all the peers information from the PeerList p
func (p PeerList) Peers() []NodeInfo {
	peers := make([]NodeInfo, 0, len(p.peers)*2)
	for _, l := range p.peers {
		for _, info := range l {
			peers = append(peers, info)
		}
	}
	return peers
}

// Ups return all the peers that are marked up from the PeerList p
func (p PeerList) Ups() []NodeInfo {
	a := p.Peers()
	n := 0
	for _, x := range a {
		if x.Up {
			a[n] = x
			n++
		}
	}
	a = a[:n]
	return a
}
