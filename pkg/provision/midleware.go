package provision

import (
	"context"

	"github.com/threefoldtech/zbus"
)

type (
	zbusKey       struct{}
	owerCacheKey  struct{}
	zdbMappingKey struct{}
)

// WithZBus adds a zbus client middleware to context
func WithZBus(ctx context.Context, client zbus.Client) context.Context {
	return context.WithValue(ctx, zbusKey{}, client)
}

// GetZBus gets a zbus client from context
func GetZBus(ctx context.Context) zbus.Client {
	value := ctx.Value(zbusKey{})
	if value == nil {
		panic("no tnodb middleware associated with context")
	}

	return value.(zbus.Client)
}

// OwnerCache interface
type OwnerCache interface {
	OwnerOf(reservationID string) (string, error)
}

// WithOwnerCache adds the owner cache to context
func WithOwnerCache(ctx context.Context, cache OwnerCache) context.Context {
	return context.WithValue(ctx, owerCacheKey{}, cache)
}

// GetOwnerCache gets the owner cache from context
func GetOwnerCache(ctx context.Context) OwnerCache {
	value := ctx.Value(owerCacheKey{})
	if value == nil {
		panic("no reservation cache associated with context")
	}

	return value.(OwnerCache)
}
