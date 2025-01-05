package rpc

import (
	"fmt"
)

func (s *Service) StorageMetrics(arg any, reply *PoolMetricsResult) error {
	pools, err := s.storageStub.Metrics(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to get pools: %w", err)
	}

	for _, pool := range pools {
		reply.Pools = append(reply.Pools, PoolMetrics{
			Name: pool.Name,
			Type: pool.Type.String(),
			Size: uint64(pool.Size),
			Used: uint64(pool.Used),
		})
	}

	return nil
}
