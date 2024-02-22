package engine

import (
	"context"
	"fmt"
)

var (
	ErrActionNotFound    = fmt.Errorf("action not found")
	ErrTypeUnknown       = fmt.Errorf("type unknown")
	ErrObjectNotFound    = fmt.Errorf("object not found")
	ErrObjectInvalidType = fmt.Errorf("invalid object type")
	ErrSpaceNotFound     = fmt.Errorf("space not found")
	ErrActionNotAllowed  = fmt.Errorf("action not allowed")
)

/// Getters

type storeKey struct{}

// GetStore returns store from engine context
func GetStore(ctx context.Context) ScopedStore {
	return ctx.Value(storeKey{}).(ScopedStore)
}

func withStore(ctx context.Context, store ScopedStore) context.Context {
	return context.WithValue(ctx, storeKey{}, store)
}

type spaceKey struct{}

// GetSpaceID gets the current space for the inflight request
func GetSpaceID(ctx context.Context) string {
	return ctx.Value(spaceKey{}).(string)
}

func withSpaceID(ctx context.Context, space string) context.Context {
	return context.WithValue(ctx, spaceKey{}, space)
}

type userKey struct{}

// GetUserID gets the in flight user id for the inflight request
func GetUserID(ctx context.Context) UserID {
	return ctx.Value(userKey{}).(UserID)
}

func withUserID(ctx context.Context, user UserID) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

type resourceKey struct{}

// GetObjectID returns the current resource name for the inflight request
func GetObjectID(ctx context.Context) string {
	return ctx.Value(resourceKey{}).(string)
}

func withObjectID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, resourceKey{}, id)
}

type existsKey struct{}

// Exists return true if current resource already exists.
func Exists(ctx context.Context) bool {
	return ctx.Value(existsKey{}).(bool)
}

func withExists(ctx context.Context, exists bool) context.Context {
	return context.WithValue(ctx, existsKey{}, exists)
}

// Request is an engine request
type Request struct {
	Type  string `json:"type"`
	User  UserID `json:"user"`
	Space string `json:"space"`

	ObjectRequest
}

// Engine Response object
type Response struct {
	ObjectResponse
}

/*
*
Engine! is the main entry point for this module. Its main functionality is to
keep a set of resources and expose their (public) functionality.

Once a resource is registered any resource function can be called knowing it's
name and input data
*/
type Engine struct {
	store Store
	types map[string]Type
}

func (e *Engine) Do(ctx context.Context, request Request) (response Response, err error) {
	// injection of higher level request data like the store for example
	exists, err := e.store.SpaceExists(request.User, request.Space)
	if err != nil {
		return response, err
	}

	if !exists {
		return response, ErrObjectNotFound
	}

	exists, typ, err := e.store.ResourceExists(request.User, request.Space, request.ResourceID)
	if err != nil {
		return response, err
	}

	if exists && typ != request.Type {
		return response, fmt.Errorf("resource '%s' exists but it's not of type '%s': %w", request.ResourceID, request.Type, ErrObjectInvalidType)
	}

	scoped := e.store.Scoped(request.User, request.Space)

	ctx = withExists(ctx, exists)
	ctx = withUserID(ctx, request.User)
	ctx = withSpaceID(ctx, request.Space)
	ctx = withStore(ctx, scoped)

	resource, ok := e.types[request.Type]
	if !ok {
		return response, ErrTypeUnknown
	}

	result, err := resource.Do(ctx, request.ObjectRequest)
	if err != nil {
		return response, err
	}

	return Response{result}, nil
}

func (e *Engine) Type(typ Type) {
	if _, ok := e.types[typ.name]; ok {
		panic("resource with same type already registered")
	}

	e.types[typ.name] = typ
}
