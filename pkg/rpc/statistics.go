package rpc

import (
	"fmt"
)

func (s *Service) Statistics(arg any, reply *Counters) error {
	stats, err := s.statisticsStub.GetCounters(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to get diagnostics: %w", err)
	}
	return convert(stats, reply)
}
