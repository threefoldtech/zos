package client

import (
	"fmt"
	"net/url"

	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/directory"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/phonebook"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
)

// Client structure
type Client struct {
	Phonebook Phonebook
	Directory Directory
	Workloads Workloads
}

// Directory API interface
type Directory interface {
	FarmRegister(farm directory.TfgridDirectoryFarm1) (schema.ID, error)
	FarmList(tid schema.ID, page *Pager) (farms []directory.TfgridDirectoryFarm1, err error)
	FarmGet(id schema.ID) (farm directory.TfgridDirectoryFarm1, err error)

	NodeRegister(node directory.TfgridDirectoryNode2) error
	NodeList(filter NodeFilter) (nodes []directory.TfgridDirectoryNode2, err error)
	NodeGet(id string, proofs bool) (node directory.TfgridDirectoryNode2, err error)

	NodeSetInterfaces(id string, ifaces []directory.TfgridDirectoryNodeIface1) error
	NodeSetPorts(id string, ports []uint) error
	NodeSetPublic(id string, pub directory.TfgridDirectoryNodePublicIface1) error

	//TODO: this method call uses types from zos that is not generated
	//from the schema. Which is wrong imho.
	NodeSetCapacity(
		id string,
		resources directory.TfgridDirectoryNodeResourceAmount1,
		dmiInfo dmi.DMI,
		disksInfo capacity.Disks,
		hypervisor []string,
	) error

	NodeUpdateUptime(id string, uptime uint64) error
	NodeUpdateUsedResources(id string, amount directory.TfgridDirectoryNodeResourceAmount1) error
}

// Phonebook interface
type Phonebook interface {
	Create(user phonebook.TfgridPhonebookUser1) (schema.ID, error)
	List(name, email string, page *Pager) (output []phonebook.TfgridPhonebookUser1, err error)
	Get(id schema.ID) (phonebook.TfgridPhonebookUser1, error)
	// Update() #TODO
	Validate(id schema.ID, message, signature string) (bool, error)
}

// Workloads interface
type Workloads interface {
	Create(reservation workloads.TfgridWorkloadsReservation1) (id schema.ID, err error)
	List(page *Pager) (reservation []workloads.TfgridWorkloadsReservation1, err error)
	Get(id schema.ID) (reservation workloads.TfgridWorkloadsReservation1, err error)

	SignProvision(id schema.ID, user schema.ID, signature string) error
	SignDelete(id schema.ID, user schema.ID, signature string) error

	Workloads(nodeID string, from uint64) ([]workloads.TfgridWorkloadsReservationWorkload1, error)
	WorkloadGet(gwid string) (result workloads.TfgridWorkloadsReservationWorkload1, err error)
	WorkloadPutResult(nodeID, gwid string, result workloads.TfgridWorkloadsReservationResult1) error
	WorkloadPutDeleted(nodeID, gwid string) error
}

// Pager for listing
type Pager struct {
	p int
	s int
}

func (p *Pager) apply(v url.Values) {
	if p == nil {
		return
	}

	if p.p < 1 {
		p.p = 1
	}

	if p.s == 0 {
		p.s = 10
	}

	v.Set("page", fmt.Sprint(p.p))
	v.Set("size", fmt.Sprint(p.s))
}

// Page returns a pager
func Page(page, size int) *Pager {
	return &Pager{p: page, s: size}
}

// NewClient creates a new client
func NewClient(u string) (*Client, error) {
	h, err := newHTTPClient(u)
	if err != nil {
		return nil, err
	}
	cl := &Client{
		Phonebook: &httpPhonebook{h},
		Directory: &httpDirectory{h},
		Workloads: &httpWorkloads{h},
	}

	return cl, nil
}
