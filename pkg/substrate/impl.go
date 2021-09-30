package substrate

import (
	"fmt"
	"net"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	//ErrInvalidVersion is returned if version 4bytes is invalid
	ErrInvalidVersion = fmt.Errorf("invalid version")
	//ErrUnknownVersion is returned if version number is not supported
	ErrUnknownVersion = fmt.Errorf("unknown version")
	//ErrNotFound is returned if an object is not found
	ErrNotFound = fmt.Errorf("object not found")
)

// Versioned base for all types
type Versioned struct {
	Version uint32
}

type Conn = *gsrpc.SubstrateAPI
type Meta = *types.Metadata

type Pool interface {
	Get() (Conn, Meta, error)
}

type poolImpl struct {
	cl *gsrpc.SubstrateAPI
}

func NewPool(url string) (Pool, error) {
	cl, err := gsrpc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}

	return &poolImpl{
		cl: cl,
	}, nil
}

// Get implements Pool interface
func (p *poolImpl) Get() (Conn, Meta, error) {
	// right now this pool implementation just tests the connection
	// makes sure that it is still active, otherwise, tries again
	// until the connection is restored.
	// A better pool implementation can be done later were multiple connections
	// can be handled
	// TODO: thread safety!
	for {
		meta, err := p.cl.RPC.State.GetMetadataLatest()
		if errors.Is(err, net.ErrClosed) {
			log.Debug().Msg("reconnecting")
			continue
		} else if err != nil {
			return nil, nil, err
		}

		return p.cl, meta, nil
	}
}

// Substrate client
type Substrate struct {
	pool Pool
}

// NewSubstrate creates a substrate client
func NewSubstrate(url string) (*Substrate, error) {
	pool, err := NewPool(url)
	if err != nil {
		return nil, err
	}

	return &Substrate{
		pool: pool,
	}, nil
}

func (s *Substrate) getVersion(b types.StorageDataRaw) (uint32, error) {
	var ver Versioned
	if err := types.DecodeFromBytes(b, &ver); err != nil {
		return 0, errors.Wrapf(ErrInvalidVersion, "failed to load version (reason: %s)", err)
	}

	return ver.Version, nil
}
