package utils

import (
	"context"
	"os"
	"os/signal"
)

// WithSignal return a context that is canceld if any of the specified signals was
// received
func WithSignal(ctx context.Context, sig ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan os.Signal)

	signal.Notify(ch, sig...)

	go func() {
		<-ch
		cancel()
		signal.Stop(ch)
	}()

	return ctx, cancel
}

// OnDone registers a callback on a context when it's done
// The ctx.Err() is passed as is to the callback function
func OnDone(ctx context.Context, cb func(error)) {
	go func() {
		<-ctx.Done()
		cb(ctx.Err())
	}()
}
