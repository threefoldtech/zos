/*
*
Engine package defines the provision engine interface and implementation code to support multiple "primitives" that
can be grouped, created, updated, and deleted

The engine requires to implement the following:
  - Create a Project. A project maps to a deployment contract for the user.
  - User need to create primitives under a certain project. Projects are isolated from other user projects. And resources are not shared
    between them
  - Q: support moving resources from a project to another ?
  - Each primitive need to support the following set of functionality
  - create() <- requires "create data" which is different from other primitives
  - update() <- requires workload id, and update data that can be different from the create data
    since sometimes can update specific parts of a
  - delete() <- deletes a resource.
  - Basic functionality above need to be consistent with dependency. A create of a resource A that relies on source B
    also means that Resource B cannot be deleted before resource A. The resource dependency is enforced by the engine
    itself, not by user checks. So once a delete call is done to the implementation layer. It should be 100% sure is safe
    to delete itself. No more checks to be done.
  - This means that a call to "create" does not only require configuration, but also dependency information. A dependency
    can then allow shared or exclusive access. For example a disk can only be used once, but a network can be used by multiple VMs.
  - For each workload type. A set of extra runtime actions should be supported.
  - For example, a "cycle" action is supported by a VM to restart itself.
  - Attach disk? detach disk? etc...
  - Some updates need to notify all dependent resources. A change to disk size need to trigger an event at the Vm to take action for example
    this is going to be tricky to implement properly
*/
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

// BaseResource is a helper resource that can be used to abstract access
// to resource storages
type BaseResource[R any] struct{}

// Current returns the current associated resource
func (r BaseResource[R]) Current(ctx Context) (R, error) {
	store := ctx.Store()

	var resource R

	if !ctx.Exists() {
		return resource, ErrObjectDoesNotExist
	}

	id := ctx.Object()

	record, err := store.RecordGet(id)
	if err != nil {
		return resource, err
	}

	// sanity check
	if record.Type != reflect.TypeFor[R]().Name() {
		return resource, ErrObjectInvalidType
	}

	if err := json.Unmarshal(record.Data, &resource); err != nil {
		return resource, fmt.Errorf("failed to decode resource %s as type %s: %w", id, record.Type, err)
	}

	return resource, nil
}

// Update the current resource to resource
func (r BaseResource[R]) Set(ctx Context, resource R) error {
	bytes, err := json.Marshal(resource)
	if err != nil {
		return err
	}

	store := ctx.Store()
	return store.RecordSet(bytes)
}

// AddDependency tries to reserve another resource in the same space as current resource. the resource is reserved temporary during the entire execution
// of the scope. If scope returns successfully the resource is reserved forever for the current resource.
// A call to RemoveDependency can then be used to release a resource
//
// probably we need ability to reserve multiple resources in one go.
func (r BaseResource[R]) AddDependency(ctx context.Context, resource string, scope func(inner context.Context) error) error {
	panic("not implemented")
}

func (r BaseResource[R]) RemoveDependency(ctx context.Context, resource string, scope func(inner context.Context) error) error {
	panic("not implemented")
}

type ResourceRequest struct {
	Action     string          `json:"action"`
	ResourceID string          `json:"resource"`
	Payload    json.RawMessage `json:"payload"`
}

type ResourceResponse struct {
	// TODO: add context to error (codes, etc)
	Error   string          `json:"error"`
	Payload json.RawMessage `json:"payload"`
}

// resourceLock is used internally to
// manage exclusive access to services
// (functions that can't be executed in parallel)
// against
type resourceGuard struct {
	sync.RWMutex
	count uint32
}

func (g *resourceGuard) Enter() {
	g.count += 1
}

func (g *resourceGuard) Exist() bool {
	g.count -= 1
	return g.count == 0
}

type Resource struct {
	name string
	flag ResourceFlag
	// all available service on this resource type
	actions map[string]Service

	// synchronize access to local resource
	// this is to make sure actions has exclusive
	// access to this resource.
	//
	// Note: since there is one instance of the engine
	// always managing resources on a single node
	// then this should be fine to prevent race conditions
	// of course this is not a good solution if resource can
	// be modified from different nodes. luckily it's not
	guards map[string]*resourceGuard
	m      sync.Mutex
}

// Do maps the request to the proper action by the time this is called the context
// already have all request related values that can be accessed via the
func (t *Resource) call(ctx *engineContext, call ResourceRequest) (response ResourceResponse, err error) {
	service, ok := t.actions[call.Action]
	if !ok {
		return response, ErrActionNotFound
	}

	// full qualified name
	id := fmt.Sprintf("%d/%s/%s", ctx.user, ctx.space, ctx.object)
	t.m.Lock()
	guard, ok := t.guards[id]
	if !ok {
		guard = &resourceGuard{}
		t.guards[id] = guard
	}
	// need to happen here before releasing the map lock
	guard.Enter()
	t.m.Unlock()

	// before we return we need to clean up the guard
	// delete it if not used anymore by anyone
	defer func(id string) {
		t.m.Lock()
		defer t.m.Unlock()

		guard := t.guards[id]
		if guard.Exist() {
			delete(t.guards, id)
		}
	}(id)

	// it's now safe to lock the guard before
	// processing
	// an exclusive service (method) require total lock
	// this also implied by MustNotExist flag to avoid double creation
	if service.flags.Is(ServiceExclusive | ServiceObjectMustNotExist) {
		guard.Lock()
		defer guard.Unlock()
	} else {
		guard.RLock()
		defer guard.RUnlock()
	}

	// if the resource already exist but the service require that
	// no resource exists with that name then we need to return an error
	exists := ctx.Exists()
	if exists && service.flags.Is(ServiceObjectMustNotExist) {
		return response, ErrObjectExists
	} else if !exists && service.flags.Is(ServiceObjectMustExists) {
		return response, ErrObjectDoesNotExist
	}

	output, err := service.Call(ctx, call.Payload)
	if err != nil {
		response.Error = err.Error()
		return
	} else {
		response.Payload = output
	}

	return response, nil
}

type ResourceBuilder struct {
	name    string
	flag    ResourceFlag
	actions map[string]Service
}

// Build a Resource for concrete type R.
func NewResourceBuilder[R any](flags ...ResourceFlag) *ResourceBuilder {

	var f ResourceFlag
	if len(flags) == 1 {
		f = flags[0]
	} else if len(flags) > 1 {
		panic("flags must be provided only once. use bitwise or to or them")
	}

	return &ResourceBuilder{
		name:    reflect.TypeFor[R]().Name(),
		flag:    f,
		actions: make(map[string]Service),
	}
}

func (t *ResourceBuilder) Action(name string, action IntoService, flags ...ServiceFlag) *ResourceBuilder {
	service := action.Into(flags...)
	if _, ok := t.actions[name]; ok {
		panic("action already exists")
	}

	t.actions[name] = service
	return t
}

func (t *ResourceBuilder) IntoResource() Resource {
	return Resource{
		name:    t.name,
		flag:    t.flag,
		actions: t.actions,
		guards:  make(map[string]*resourceGuard),
	}
}

type ResourceFlag uint8

const (
	// A resource is exclusive if it can only be used by only one other resource
	// (like a disk) and can't be shared between multiple resources as a dependency
	ResourceExclusive ResourceFlag = 1 << iota
)

func (s ResourceFlag) Is(f ResourceFlag) bool {
	return s&f == f
}
