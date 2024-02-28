package engine

import (
	"context"
	"fmt"
	"slices"
)

var (
	ErrActionNotFound     = fmt.Errorf("action not found")
	ErrResourceUnknown    = fmt.Errorf("resource unknown")
	ErrObjectInvalidType  = fmt.Errorf("invalid object type")
	ErrSpaceNotFound      = fmt.Errorf("space not found")
	ErrObjectExists       = fmt.Errorf("object exists")
	ErrObjectDoesNotExist = fmt.Errorf("object does not exist")
	ErrObjectInUse        = fmt.Errorf("object is in use")
	ErrObjectNotUsed      = fmt.Errorf("object is not used")
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
		engine: e,
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

// use is an internal function that is used mainly to lock resources before they
// can be added as a dependency to a master resource.
// on cb success the dependencies will updated successfully with their new master.
// on error, no changes are made. It's up to the cb to make sure to revert any changes to reality
// if failed. It also has to be idempotent to make sure rerunning the same function does not change
// or suddenly start giving errors.
// the function also take care of resource exclusivity
func (e *Engine) use(ctx Context, cb Use, resources ...string) error {
	// reserve holds resources inside a space for during the execution time
	// of the call back. Then commit dependency once the function returns successfully

	// todo: what if entire process crashed before committing but after the dependency
	// was physically used by the parent resource.

	var guards []Guard
	defer func() {
		for _, g := range guards {
			g.Unlock()
			g.Exit()
		}
	}()

	var records []Record
	// hold exclusive access to resources by their type.
	for _, resource := range resources {
		if resource == ctx.Object() {
			return fmt.Errorf("cyclic dependency detected")
		}

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

		// we do another get after acquiring the lock to make sure
		// the object still exists while we are holding the lock.
		// might be good also to provide the record as is to the cb function
		record, err = e.store.RecordGet(ctx.User(), ctx.Space(), resource)
		if err != nil {
			return fmt.Errorf("resource %s: %w", resource, err)
		}

		if typ.flag.Is(ResourceExclusive) {
			// TODO
			// If resource is an exclusive resource it means the resource
			// can't be a dependency to multiple master resources at the same
			// time. For example multiple VMs can't use the same disk, but they
			// can use the same network for example.
			if len(record.Masters) >= 1 {
				return fmt.Errorf("resource %s: %w", resource, ErrObjectInUse)
			}
		}

		records = append(records, record)
	}

	if err := cb(ctx, records); err != nil {
		// cb need to be sure to completely revert on error
		return err
	}

	// commit dependency
	for _, resource := range resources {
		if err := e.store.MasterAdd(ctx.User(), ctx.Space(), resource, ctx.Object()); err != nil {
			// TODO: this should only be a db error. the question is if this is fatal
			return fmt.Errorf("failed to add resource %s as dependency: %w", resource, err)
		}
	}

	return nil
}

// opposite of use
func (e *Engine) unUse(ctx Context, cb Use, resources ...string) error {
	var guards []Guard
	defer func() {
		for _, g := range guards {
			g.Unlock()
			g.Exit()
		}
	}()

	var records []Record
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

		// we do another get after acquiring the lock to make sure
		// the object still exists while we are holding the lock.
		// might be good also to provide the record as is to the cb function
		record, err = e.store.RecordGet(ctx.User(), ctx.Space(), resource)
		if err != nil {
			return fmt.Errorf("resource %s: %w", resource, err)
		}

		used := slices.ContainsFunc(record.Masters, func(m string) bool {
			return m == ctx.Object()
		})

		if !used {
			// object is not used by current resource
			return fmt.Errorf("resource %s: %w", resource, ErrObjectNotUsed)
		}

		records = append(records, record)
	}

	if err := cb(ctx, records); err != nil {
		// cb need to be sure to completely revert on error
		return err
	}

	// commit dependency
	for _, resource := range resources {
		if err := e.store.MasterRemove(ctx.User(), ctx.Space(), resource, ctx.Object()); err != nil {
			// TODO: this should only be a db error. the question is if this is fatal
			return fmt.Errorf("failed to add resource %s as dependency: %w", resource, err)
		}
	}

	return nil
}
