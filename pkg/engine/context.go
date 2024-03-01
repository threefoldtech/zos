package engine

import "context"

// Use is a call back for Context.Use function that reserves
//
// During the execution of this callback it's granted that
//   - All requested resources are exclusively locked
//   - exclusive Resource types (that can't be shared) are granted to be
//     unused by other resources
//
// On successful return of the call back resources master is updated
type Use func(Context, []Record) error
type Context interface {
	// Ctx returns current request context
	Ctx() context.Context
	// Space returns current space id
	Space() string
	// User returns current user id
	User() UserID
	// Object returns current object id
	Object() string
	// Exists returns if object already exists
	// in store. Normally false on first call that
	// creates the resource
	Exists() bool
	// Store returns store
	Store() ScopedStore

	// Use runs cb resource while locking resources.
	// On cb success, the resources are assigned forever
	// to their master
	Use(cb Use, resources ...string) error
	// UnUse runs cb resource while locking resources.
	// On cb success, the resources are released as a
	// dependency to current resource
	UnUse(cb Use, resources ...string) error
}

type engineContext struct {
	ctx    context.Context
	space  string
	user   UserID
	object string
	exists bool
	typ    string
	engine *Engine
}

// Ctx returns current request context
func (ctx *engineContext) Ctx() context.Context {
	return ctx.ctx
}

// Space returns current space id
func (ctx *engineContext) Space() string {
	return ctx.space
}

// User returns current user id
func (ctx *engineContext) User() UserID {
	return ctx.user
}

// Object returns current object id
func (ctx *engineContext) Object() string {
	return ctx.object
}

// Exists returns if object already exists
// in store. Normally false on first call that
// creates the resource
func (ctx *engineContext) Exists() bool {
	return ctx.exists
}

// Store gives scoped (limited) access to store to get/set current
// resource data and also inspect other resource in the same space
func (ctx *engineContext) Store() ScopedStore {
	return ctx.engine.store.Scoped(ctx.user, ctx.space, ctx.object, ctx.typ)
}

// Use runs callback `use` while holding exclusive lock to resources
// it verifies that resources exist and also are not used by other
// resources (if they are of exclusive resource type)
// If `use` callback was successful the resources are assigned their new master
// to have this current resource
//
// **NOTE** the callback must be idempotent. It also need to revert any
// changes to reality it made in case of error so the system does not end up
// in an inconsistent situation
func (ctx *engineContext) Use(use Use, resources ...string) error {
	return ctx.engine.use(ctx, use, resources...)
}

// UnUse exactly like Use but should remove the dependency. Same rules apply. It also
// remove the current resource as master to the resources supplied
func (ctx *engineContext) UnUse(use Use, resources ...string) error {
	return ctx.engine.unUse(ctx, use, resources...)
}
