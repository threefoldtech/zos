package engine

import (
	"context"
	"fmt"
)

var (
	ErrActionNotFound     = fmt.Errorf("action not found")
	ErrResourceUnknown    = fmt.Errorf("resource unknown")
	ErrObjectInvalidType  = fmt.Errorf("invalid object type")
	ErrSpaceNotFound      = fmt.Errorf("space not found")
	ErrObjectExists       = fmt.Errorf("object exists")
	ErrObjectDoesNotExist = fmt.Errorf("object does not exist")
)

// Request is an engine request
type Request struct {
	Type  string `json:"type"`
	User  UserID `json:"user"`
	Space string `json:"space"`

	ResourceRequest
}

// Engine Response object
type Response struct {
	ResourceResponse
}

/*
*
Engine! is the main entry point for this module. Its main functionality is to
keep a set of resources and expose their (public) functionality.

Once a resource is registered any resource function can be called knowing it's
name and input data
*/
type Engine struct {
	store     Store
	resources map[string]*Resource
}

func (e *Engine) Do(ctx context.Context, request Request) (response Response, err error) {
	// injection of higher level request data like the store for example
	exists, err := e.store.SpaceExists(request.User, request.Space)
	if err != nil {
		return response, err
	}

	if !exists {
		return response, ErrObjectDoesNotExist
	}

	exists, typ, err := e.store.RecordExists(request.User, request.Space, request.ResourceID)
	if err != nil {
		return response, err
	}

	if exists && typ != request.Type {
		return response, fmt.Errorf("resource '%s' exists but it's not of type '%s': %w", request.ResourceID, request.Type, ErrObjectInvalidType)
	}

	scoped := e.store.Scoped(request.User, request.Space, request.ResourceID, typ)

	engineCtx := engineContext{
		ctx:    ctx,
		space:  request.Space,
		user:   request.User,
		object: request.ResourceID,
		exists: exists,
		store:  scoped,
	}

	resource, ok := e.resources[request.Type]
	if !ok {
		return response, ErrResourceUnknown
	}

	result, err := resource.call(&engineCtx, request.ResourceRequest)
	if err != nil {
		return response, err
	}

	return Response{result}, nil
}

func (e *Engine) Resource(typ *Resource) {
	if _, ok := e.resources[typ.name]; ok {
		panic("resource with same type already registered")
	}

	e.resources[typ.name] = typ
}
