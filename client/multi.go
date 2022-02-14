package client

import (
	"context"
	"encoding/json"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/rmb"
)

// NodeClient struct
type MultiNodeClient struct {
	twins []uint32
	bus   rmb.MultiDestinationClient
}

func NewMultiNodeClient(twins []uint32, bus rmb.MultiDestinationClient) MultiNodeClient {
	return MultiNodeClient{twins, bus}
}

type CountersResult struct {
	*rmb.ResponseMetadata `json:"-"`
	Total                 gridtypes.Capacity `json:"total"`
	Used                  gridtypes.Capacity `json:"used"`
}

func NewCountersResult() rmb.Response {
	return &CountersResult{
		ResponseMetadata: &rmb.ResponseMetadata{},
		Total:            gridtypes.Capacity{},
		Used:             gridtypes.Capacity{},
	}
}
func (r *CountersResult) SetResponse(bs []byte) error {
	return json.Unmarshal(bs, r)
}

// Counters returns some node statistics. Including total and available cpu, memory, storage, etc...
func (n *MultiNodeClient) Counters(ctx context.Context) (chan CountersResult, error) {
	const cmd = "zos.statistics.get"

	ch, err := n.bus.Call(ctx, n.twins, cmd, nil, NewCountersResult)
	if err != nil {
		return nil, err
	}
	ch2 := make(chan CountersResult)
	go func() {
		defer close(ch2)
		// ctx is handled inside n.bus.Call
		for elem := range ch {
			ch2 <- *elem.(*CountersResult)
		}
	}()
	return ch2, nil
}
