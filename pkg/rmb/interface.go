package rmb

import (
	"context"
	"fmt"
)

var (
	ErrFunctionNotFound = fmt.Errorf("function not found")
)

type Handler func(ctx context.Context, payload []byte) (interface{}, error)
type Middleware func(ctx context.Context, payload []byte) (context.Context, error)

type Router interface {
	WithHandler(route string, handler Handler) error
	Subroute(route string) Router
	Use(Middleware)
}
