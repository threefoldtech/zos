package rmb

import (
	"context"

	"github.com/rs/zerolog/log"
)

func LoggerMiddleware(ctx context.Context, payload []byte) (context.Context, error) {
	msg := GetMessage(ctx)
	log.Info().Uint32("twin", msg.TwinSrc).Str("fn", msg.Command).Int("size", len(payload)).Msg("call")
	return ctx, nil
}

var (
	_ Middleware = LoggerMiddleware
)
