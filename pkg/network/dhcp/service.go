package dhcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/zinit"
)

type ClientService struct {
	Name      string
	Iface     string
	Namespace string

	z *zinit.Client
}

func NewService(iface string, namespace string, z *zinit.Client) ClientService {
	return ClientService{
		Name:      fmt.Sprintf("dhcp-%s", iface),
		Iface:     iface,
		Namespace: namespace,
		z:         z,
	}
}

func (s ClientService) IsUsingOlderClient() bool {
	olderService, _ := s.z.Matches(zinit.WithName(s.Name), zinit.WithExec("udhcpc"))
	return len(olderService) > 0
}

func (s ClientService) Create() error {
	log.Info().Msgf("create %s zinit service", s.Name)

	exec := fmt.Sprintf("/usr/sbin/dhcpcd %s -B", s.Iface)
	if s.Namespace != "" {
		exec = fmt.Sprintf("ip netns exec %s %s", strings.TrimSpace(s.Namespace), exec)
	}

	err := zinit.AddService(s.Name, zinit.InitService{
		Exec:    exec,
		Oneshot: false,
		After:   []string{},
	})

	if err != nil {
		log.Error().Err(err).Msgf("fail to create %s zinit service", s.Name)
		return err
	}

	if err := s.z.Monitor(s.Name); err != nil && err != zinit.ErrAlreadyMonitored {
		log.Error().Err(err).Msgf("fail to start monitoring %s zinit service", s.Name)
		return err
	}

	return err
}

func (s ClientService) Destroy() error {
	if err := s.z.Destroy(20*time.Second, s.Name); err != nil {
		log.Error().Err(err).Msgf("fail to terminate zinit service", s.Name)
		return err
	}

	return nil
}

func (s ClientService) DestroyOlderService() error {
	if s.IsUsingOlderClient() {
		if err := s.Destroy(); err != nil {
			log.Error().Err(err).Msgf("fail to terminate older %s (udhcpc) service", s.Name)
			return err
		}
	}

	return nil
}

func (s ClientService) Start() error {
	return s.z.Start(s.Name)
}

func (s ClientService) Stop() error {
	return s.z.Stop(s.Name)
}
