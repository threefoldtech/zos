package rmb

import (
	"context"
	"fmt"
)

var (
	// ErrFunctionNotFound is an err returned if the handler function is not found
	ErrFunctionNotFound = fmt.Errorf("function not found")
)

// Handler is a handler function type
type Handler func(ctx context.Context, payload []byte) (interface{}, error)

// Middleware is middleware function type
type Middleware func(ctx context.Context, payload []byte) (context.Context, error)

// Router is the router interface
type Router interface {
	WithHandler(route string, handler Handler)
	Subroute(route string) Router
	Use(Middleware)
}

type Response interface {
	SetError(err error)
	SetTwin(twin uint32)
	Error() error
	Twin() uint32

	SetResponse(bs []byte) error
}

// Client is an rmb abstract client interface.
type Client interface {
	Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error
}

type MultiDestinationClient interface {
	Call(ctx context.Context, twins []uint32, fn string, data interface{}, constructor func() Response) (chan Response, error)
}
