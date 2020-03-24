package client

import (
	"fmt"
	"net/url"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/phonebook"
)

// Client structure
type Client struct {
	Phonebook Phonebook
	Directory Directory
	// Reservations Reservation
}

// Directory API interface
type Directory interface {
	FarmRegister()
	FarmList()
	FarmGet()

	NodeRegister()
	NodeList()
	NodeGet()

	NodeSetInterfaces()
	NodeSetPorts()
	NodeSetPublic()
	NodeSetCapacity()

	NodeUpdateUptime()
	NodeUpdatedUptime()
}

// Phonebook interface
type Phonebook interface {
	Create(user phonebook.TfgridPhonebookUser1) (schema.ID, error)
	List(name, email string, page *Pager) (output []phonebook.TfgridPhonebookUser1, err error)
	Get(id schema.ID) (phonebook.TfgridPhonebookUser1, error)
	// Update()
	Validate(id schema.ID, message, signature string) (bool, error)
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

func NewClient(u string) (*Client, error) {
	h, err := newHTTPClient(u)
	if err != nil {
		return nil, err
	}
	cl := &Client{
		Phonebook: &httpPhonebook{h},
	}

	return cl, nil
}
