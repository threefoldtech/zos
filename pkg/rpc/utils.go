package rpc

import "context"

type twinId struct{}

func GetTwinFromIp(ip string) uint32 {
	// placeholder, waiting for implementation in mycelium
	return 0
}

func SetTwinId(ctx context.Context, id uint32) context.Context {
	return context.WithValue(ctx, twinId{}, id)
}

func GetTwinID(ctx context.Context) uint32 {
	twin, ok := ctx.Value(twinId{}).(uint32)
	if !ok {
		panic("failed to load twin id from context")
	}

	return twin
}
