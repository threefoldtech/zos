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
)

var (
	ErrorResourceNotFound    = fmt.Errorf("not found")
	ErrorResourceInvalidType = fmt.Errorf("invalid type")
)

type Exclusive struct{}
type Inclusive struct{}

// Resource of R is a resource base implementation that allows
// seamless integration with the engine internals and storage
type Resource[R any, P Exclusive | Inclusive] struct{}

func (Resource[R, P]) IsExclusive() bool {
	return reflect.TypeFor[P]() == reflect.TypeOf(Exclusive{})
}

func (r Resource[R, P]) Name() string {
	return reflect.TypeFor[R]().Name()
}

// Current returns the current associated resource
func (r Resource[R, P]) Current(ctx context.Context) (R, error) {
	store := GetStore(ctx)
	id, ok := GetResourceID(ctx)

	var resource R
	if !ok {
		return resource, ErrorResourceNotFound
	}

	typ, bytes, err := store.ResourceGet(id)
	if err != nil {
		return resource, err
	}

	// sanity check
	if typ != reflect.TypeFor[R]().Name() {
		return resource, ErrorResourceInvalidType
	}

	if err := json.Unmarshal(bytes, &resource); err != nil {
		return resource, fmt.Errorf("failed to decode resource %s as type %s: %w", id, typ, err)
	}

	return resource, nil
}

// Update the current resource to resource
func (r Resource[R, P]) Set(ctx context.Context, resource R) error {
	bytes, err := json.Marshal(resource)
	if err != nil {
		return err
	}

	store := GetStore(ctx)
	return store.ResourceSet(bytes)
}

// AddDependency tries to reserve another resource in the same space as current resource. the resource is reserved temporary during the entire execution
// of the scope. If scope returns successfully the resource is reserved forever for the current resource.
// A call to RemoveDependency can then be used to release a resource
func (r Resource[R, P]) AddDependency(ctx context.Context, resource string, scope func(inner context.Context) error) error {
	panic("not implemented")
}

func (r Resource[R, P]) RemoveDependency(ctx context.Context, resource string, scope func(inner context.Context) error) error {
	panic("not implemented")
}
