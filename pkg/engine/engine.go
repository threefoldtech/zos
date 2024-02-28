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
Engine! is the main entry point for this module. Its main functionality is to
keep a set of resources and expose their (public) functionality.

Once a resource is registered any resource function can be called knowing it's
name and input data
*/
type Engine struct {
	store     Store
	resources map[string]*Resource
	guard     *AccessGuard
}

func (e *Engine) Handle(ctx context.Context, request Request) (response Response, err error) {
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

func (e *Engine) Resource(typ *Resource) *Engine {
	if _, ok := e.resources[typ.name]; ok {
		panic("resource with same type already registered")
	}

	e.resources[typ.name] = typ

	return e
}

// hold is an internal function that
func (e *Engine) hold(ctx Context, cb func(Context) error, resources ...string) error {
	// reserve holds resources inside a space for during the execution time
	// of the call back. Then commit dependency once the function returns successfully

	// todo: what if entire process crashed before committing but after the dependency
	// was physically used by the parent resource.

	id := fmt.Sprintf("%d/%s", ctx.User(), ctx.Space())
	guard := e.guard.Enter(id)
	defer guard.Exit()

	guard.Lock()
	defer guard.Unlock()

	var guards []Guard
	defer func() {
		for _, g := range guards {
			g.Unlock()
			g.Exit()
		}
	}()

	// hold exclusive access to resources by their type.
	for _, resource := range resources {
		record, err := e.store.RecordGet(ctx.User(), ctx.Space(), resource)
		if err != nil {
			return fmt.Errorf("resource %s: %w", resource, err)
		}

		typ, ok := e.resources[record.Type]
		if !ok {
			// This is totally shouldn't be possible if an object
			// is in store its type should exist.
			return fmt.Errorf("resource %s: %w", resource, ErrObjectInvalidType)
		}

		// hold guard on that specific resource
		// note that holding the guard does not relate to if the resource object
		// actually exists or not or what its status is. It's a mechanism that
		// forces the engine to block actions on that resource until this process
		// is finished.
		guard := typ.getGuard(ctx, resource)
		guard.Lock()

		guards = append(guards, guard)

		// Get the record now that should be in consistent state
	}

	return nil
}
