package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
)

type Yggdrasil struct {
	connectionURI string
	conn          net.Conn
}

type response struct {
	Status   string                 `json:"status,omitempty"`
	Request  map[string]interface{} `json:"request,omitempty"`
	Response json.RawMessage        `json:"response,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

func NewYggdrasil(connectionURI string) *Yggdrasil {
	return &Yggdrasil{
		connectionURI: connectionURI,
	}
}

func (y *Yggdrasil) Connect() error {
	u, err := url.Parse(y.connectionURI)
	if err != nil {
		return fmt.Errorf("wrong format for connection URI %v: %w", y.connectionURI, err)
	}

	conn, err := net.Dial(strings.ToLower(u.Scheme), u.Host)
	if err != nil {
		return err
	}

	y.conn = conn
	return nil
}

func (y *Yggdrasil) Close() error {
	if y.conn != nil {
		return y.conn.Close()
	}
	return nil
}

func (y *Yggdrasil) execReq(req string) (response, error) {
	resp := response{}
	if _, err := y.conn.Write([]byte(req)); err != nil {
		return resp, err
	}

	if err := json.NewDecoder(y.conn).Decode(&resp); err != nil {
		return resp, err
	}

	if resp.Error != "" {
		log.Printf("%s", string(resp.Response))
		return resp, fmt.Errorf(resp.Error)
	}

	return resp, nil
}

func (y *Yggdrasil) GetSelf() (nodeinfo yggdrasil.NodeInfo, err error) {

	resp, err := y.execReq(`{"keepalive":true, "request":"getSelf"}`)
	if err != nil {
		return
	}

	data := struct {
		Self map[string]yggdrasil.NodeInfo `json:"self"`
	}{}
	if err = json.Unmarshal(resp.Response, &data); err != nil {
		return
	}

	for k := range data.Self {
		nodeinfo = data.Self[k]
		nodeinfo.IPv6Addr = k
		break
	}

	return nodeinfo, nil
}

func (y *Yggdrasil) GetPeers() ([]yggdrasil.Peer, error) {
	resp, err := y.execReq(`{"keepalive":true, "request":"getpeers"}`)
	if err != nil {
		return nil, err
	}

	data := struct {
		Peers map[string]yggdrasil.Peer `json:"peers"`
	}{}
	if err = json.Unmarshal(resp.Response, &data); err != nil {
		return nil, err
	}

	peers := make([]yggdrasil.Peer, 0, len(data.Peers))

	for k := range data.Peers {
		peer := data.Peers[k]
		peer.IPv6Addr = k
		peers = append(peers, peer)
	}

	return peers, nil
}

func (y *Yggdrasil) AddPeer(uri string) ([]string, error) {
	req := fmt.Sprintf(`{"keepalive":true, "request":"addpeer", "uri":"%s"}`, uri)
	resp, err := y.execReq(req)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(resp.Response))
	added := struct {
		Added []string `json:"added"`
	}{}
	if err := json.Unmarshal(resp.Response, &added); err != nil {
		return nil, err
	}

	return added.Added, nil
}
