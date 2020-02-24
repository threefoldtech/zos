package models

import (
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// DefaultPageSize is default page size
	DefaultPageSize int64 = 100
)

// Pager is the find options wrapper
type Pager *options.FindOptions

// Page creates a Pager
func Page(p int64, size ...int64) Pager {
	ps := DefaultPageSize
	if len(size) > 0 {
		ps = size[0]
	}
	skip := p * ps
	return options.Find().SetLimit(ps).SetSkip(skip)
}
