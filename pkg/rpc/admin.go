package rpc

func (s *Service) AdminPublicNICSet(arg string, reply *any) error {
	return s.networkerStub.SetPublicExitDevice(s.ctx, arg)
}

func (s *Service) AdminPublicNICGet(arg any, reply *ExitDevice) error {
	ed, err := s.networkerStub.GetPublicExitDevice(s.ctx)
	if err != nil {
		return err
	}

	reply.AsDualInterface = ed.AsDualInterface
	reply.IsDual = ed.IsDual
	reply.IsSingle = ed.IsSingle
	return nil
}

func (s *Service) AdminInterfaces(arg any, reply *Interfaces) error {
	interfaces, err := s.networkerStub.Interfaces(s.ctx, "", "")
	if err != nil {
		return err
	}

	for name, inf := range interfaces.Interfaces {
		reply.Interfaces = append(reply.Interfaces, Interface{
			Name: name,
			Mac:  inf.Mac,
			Ips: func() []string {
				var ips []string
				for _, ip := range inf.IPs {
					ips = append(ips, ip.String())
				}
				return ips
			}(),
		})
	}

	return nil
}
