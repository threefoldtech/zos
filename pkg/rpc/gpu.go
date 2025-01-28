package rpc

func (s *Service) GpuList(arg any, reply *GPUs) error {
	gpus, err := s.statisticsStub.ListGPUs(s.ctx)
	if err != nil {
		return err
	}

	for _, gpu := range gpus {
		reply.GPUs = append(reply.GPUs, GPU{
			ID:       gpu.ID,
			Vendor:   gpu.Vendor,
			Device:   gpu.Device,
			Contract: gpu.Contract,
		})
	}

	return nil
}
