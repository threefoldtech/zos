package provision

import (
	"context"
)

type (
	storageKey struct{}
)

// GetStorage from context
func GetStorage(ctx context.Context) Storage {
	c := ctx.Value(storageKey{})
	if c == nil {
		panic("cache middleware is not set")
	}

	return c.(Storage)
}
