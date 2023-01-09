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

type Options struct {
	timeout uint32
}

func DefaultOptions() Options {
	return Options{
		timeout: 600, // 5 minutes
	}
}

func (o Options) WithTimeout(timeout uint32) Options {
	o.timeout = timeout
	return o
}

// Client is an rmb abstract client interface.
type Client interface {
	Call(ctx context.Context, twin uint32, fn string, data interface{}, result interface{}) error
}
