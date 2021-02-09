package provision

import (
	"context"
)

type (
	cacheKey struct{}
)

// GetCache from context
func GetCache(ctx context.Context) Storage {
	c := ctx.Value(cacheKey{})
	if c == nil {
		panic("cache middleware is not set")
	}

	return c.(Storage)
}
