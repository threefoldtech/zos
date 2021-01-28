package options

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
)

type sysOption struct {
	key string
	on  bool
}

func (s *sysOption) apply(inf string) error {
	_, err := sysctl.Sysctl(fmt.Sprintf(s.key, inf), flag(s.on))
	return err
}

// IPv6Disable disabled Ipv6 on interface
func IPv6Disable(f bool) Option {
	return &sysOption{
		key: "net.ipv6.conf.%s.disable_ipv6",
		on:  f,
	}
}
