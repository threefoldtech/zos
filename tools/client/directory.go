package client

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
)

type httpDirectory struct {
	*httpClient
}

func (d *httpDirectory) FarmRegister(farm directory.TfgridDirectoryFarm1) (schema.ID, error) {
	var output struct {
		ID schema.ID `json:"id"`
	}

	err := d.post(d.url("farms"), farm, &output, http.StatusCreated)
	return output.ID, err
}

func (d *httpDirectory) FarmList(tid schema.ID, page *Pager) (farms []directory.TfgridDirectoryFarm1, err error) {
	query := url.Values{}
	page.apply(query)
	if tid > 0 {
		query.Set("owner", fmt.Sprint(tid))
	}

	err = d.get(d.url("farms"), query, &farms, http.StatusOK)
	return
}

func (d *httpDirectory) FarmGet(id schema.ID) (farm directory.TfgridDirectoryFarm1, err error) {
	err = d.get(d.url("farms", fmt.Sprint(id)), nil, &farm, http.StatusOK)
	return
}

func (d *httpDirectory) NodeRegister(node directory.TfgridDirectoryNode2) error {
	return d.post(d.url("nodes"), node, nil, http.StatusCreated)
}

func (d *httpDirectory) NodeList(filter NodeFilter) (nodes []directory.TfgridDirectoryNode2, err error) {
	query := url.Values{}
	filter.Apply(query)
	err = d.get(d.url("nodes"), query, &nodes, http.StatusOK)
	return
}

func (d *httpDirectory) NodeGet(id string, proofs bool) (node directory.TfgridDirectoryNode2, err error) {
	query := url.Values{}
	query.Set("proofs", fmt.Sprint(proofs))
	err = d.get(d.url("nodes", id), query, &node, http.StatusOK)
	return
}

func (d *httpDirectory) NodeSetInterfaces() {}
func (d *httpDirectory) NodeSetPorts()      {}
func (d *httpDirectory) NodeSetPublic()     {}
func (d *httpDirectory) NodeSetCapacity()   {}

func (d *httpDirectory) NodeUpdateUptime()  {}
func (d *httpDirectory) NodeUpdatedUptime() {}
