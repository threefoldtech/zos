package utils

import (
	"context"
	"sync"
)

type Mark interface {
	Done(ctx context.Context) error
	Signal()
}

// mark is
type mark struct {
	ch chan struct{}
	o  sync.Once
}

func NewMark() Mark {
	return &mark{ch: make(chan struct{})}
}

func (m *mark) Done(ctx context.Context) error {
	select {
	case <-m.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *mark) Signal() {
	m.o.Do(func() {
		close(m.ch)
	})
}
