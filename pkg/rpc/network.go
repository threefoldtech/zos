package rpc

import (
	"fmt"
	"net"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func (s *Service) NetworkPublicIps(arg any, reply *Ips) error {
	ips, err := s.provisionStub.ListPublicIPs(s.ctx)
	if err != nil {
		return err
	}

	reply.Ips = append(reply.Ips, ips...)
	return nil
}

func (s *Service) NetworkHasIpv6(arg any, reply *bool) error {
	ipData, err := s.networkerStub.GetPublicIPv6Subnet(s.ctx)
	hasIP := ipData.IP != nil && err == nil
	*reply = hasIP
	return nil
}

func (s *Service) NetworkInterfaces(arg any, reply *Interfaces) error {
	type q struct {
		inf    string
		ns     string
		rename string
	}

	for _, i := range []q{
		{"zos", "", "zos"},
		{"nygg6", "ndmz", "ygg"},
	} {
		ips, _, err := s.networkerStub.Addrs(s.ctx, i.inf, i.ns)
		if err != nil {
			return fmt.Errorf("failed to get ips for '%s' interface: %w", i, err)
		}

		reply.Interfaces = append(reply.Interfaces, Interface{
			Name: i.rename,
			Ips: func() []string {
				var list []string
				for _, item := range ips {
					ip := net.IP(item)
					list = append(list, ip.String())
				}

				return list
			}(),
		})
	}

	return nil
}

func (s *Service) NetworkPublicConfig(arg any, reply *PublicConfig) error {
	config, err := s.networkerStub.GetPublicConfig(s.ctx)
	if err != nil {
		return err
	}

	reply.Domain = config.Domain
	reply.IPv4 = config.IPv4.String()
	reply.IPv6 = config.IPv6.String()
	reply.GW4 = config.GW4.String()
	reply.GW6 = config.GW6.String()
	reply.Type = string(config.Type)
	return nil
}

func (s *Service) NetworkWGPorts(arg any, reply *WGPorts) error {
	ports, err := s.networkerStub.WireguardPorts(s.ctx)
	if err != nil {
		return err
	}

	for _, port := range ports {
		reply.Ports = append(reply.Ports, uint64(port))
	}

	return nil
}

func (s *Service) NetworkPrivateIps(arg string, reply *Ips) error {
	ips, err := s.provisionStub.ListPrivateIPs(s.ctx, GetTwinID(s.ctx), gridtypes.Name(arg))
	if err != nil {
		return err
	}

	reply.Ips = append(reply.Ips, ips...)
	return nil
}
