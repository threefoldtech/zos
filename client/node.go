package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var (
	successCodes = []int{http.StatusOK}
)

// NodeClient is a client of the node
type NodeClient struct {
	client *Client
	ip     net.IP
}

func (n *NodeClient) response(r *http.Response, o interface{}, codes ...int) error {
	defer r.Body.Close()
	if len(codes) == 0 {
		codes = successCodes
	}

	in := func(i int, l []int) bool {
		for _, v := range l {
			if v == i {
				return true
			}
		}
		return false
	}

	if !in(r.StatusCode, codes) {
		msg, _ := ioutil.ReadAll(r.Body)
		return fmt.Errorf("invalid response (%s): %s", r.Status, string(msg))
	}

	defer func() {
		ioutil.ReadAll(r.Body)
	}()

	if o != nil {
		return json.NewDecoder(r.Body).Decode(o)
	}

	return nil
}

// Deploy sends the workload for node to deploy. On success means the node
// accepted the workload (it passed validation), doesn't mean it has been
// deployed. the user then can pull on the workload status until it passes (or fail)
func (n *NodeClient) Deploy(wl *gridtypes.Workload) (string, error) {
	wl.ID = n.client.id
	if err := wl.Sign(n.client.sk); err != nil {
		return "", errors.Wrap(err, "failed to sign the workload")
	}

	return "", nil
}
