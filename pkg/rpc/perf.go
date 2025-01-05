package rpc

func (s *Service) PerfGetCpuBench(arg any, reply *CpuBenchTaskResult) error {
	r, err := s.performanceMonitorStub.GetCpuBenchTaskResult(s.ctx)
	if err != nil {
		return err
	}

	return convert(r, reply)
}

func (s *Service) PerfGetHealth(arg any, reply *HealthTaskResult) error {
	r, err := s.performanceMonitorStub.GetHealthTaskResult(s.ctx)
	if err != nil {
		return err
	}

	return convert(r, reply)
}

func (s *Service) PerfGetIperf(arg any, reply *IperfTaskResult) error {
	r, err := s.performanceMonitorStub.GetIperfTaskResult(s.ctx)
	if err != nil {
		return err
	}

	return convert(r, reply)
}

func (s *Service) PerfGetPublicIP(arg any, reply *PublicIpTaskResult) error {
	r, err := s.performanceMonitorStub.GetPublicIpTaskResult(s.ctx)
	if err != nil {
		return err
	}

	return convert(r, reply)
}

func (s *Service) PerfGetAll(arg any, reply *AllTaskResult) error {
	r, err := s.performanceMonitorStub.GetAllTaskResult(s.ctx)
	if err != nil {
		return err
	}

	return convert(r, reply)
}
