package engine

import "context"

/// Getters

type storeKey struct{}

// GetStore returns store from engine context
func GetStore(ctx context.Context) ScopedStore {
	return ctx.Value(storeKey{}).(ScopedStore)
}

type spaceKey struct{}

// GetSpaceID gets the current space for the inflight request
func GetSpaceID(ctx context.Context) string {
	return ctx.Value(spaceKey{}).(string)
}

type userKey struct{}

// GetUserID gets the in flight user id for the inflight request
func GetUserID(ctx context.Context) UserID {
	return ctx.Value(userKey{}).(UserID)
}

type resourceKey struct{}

// this can be nil for an inflight request this is why we return also a bool
func GetResourceID(ctx context.Context) (string, bool) {
	value := ctx.Value(resourceKey{})
	if value == nil {
		return "", false
	}

	return value.(string), true
}
