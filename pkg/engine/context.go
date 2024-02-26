package engine

import "context"

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
}

type engineContext struct {
	ctx    context.Context
	space  string
	user   UserID
	object string
	exists bool
	store  ScopedStore
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

func (ctx *engineContext) Store() ScopedStore {
	return ctx.store
}
