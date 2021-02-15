package substrate

import gsrpc "github.com/centrifuge/go-substrate-rpc-client"

type substrateClient struct {
	cl *gsrpc.SubstrateAPI
}

// NewSubstrate creates a substrate client
func NewSubstrate(url string) (Substrate, error) {
	cl, err := gsrpc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}

	return &substrateClient{
		cl: cl,
	}, nil
}
