package main

import (
	"context"
	"testing"
	"time"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/primitives"

	"github.com/rs/zerolog/log"
)

func TestWSSource(t *testing.T) {
	source := provision.NewWSSource(
		"HT9yNvHRQs65dyRJeztAdEnXKweCX2bhgvD9i1WFPUNs",
		primitives.WorkloadToProvisionType,
		primitives.ProvisionOrder,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	s := source.Reservations(ctx)

	for r := range s {
		log.Printf("received %v\n", r.ID)
	}
}
