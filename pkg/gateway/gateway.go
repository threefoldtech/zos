package gateway

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/zinit"
	"gopkg.in/yaml.v2"
)

const (
	traefikService = "traefik"
)

type gatewayModule struct {
	proxyConfigPath string
	traefikStarted  bool
}

type ProxyConfig struct {
	Http HTTPConfig
}

type HTTPConfig struct {
	Routers  map[string]Router
	Services map[string]Service
}

type Router struct {
	Rule    string
	Service string
}

type Service struct {
	LoadBalancer LoadBalancer
}

type LoadBalancer struct {
	Servers []Server
}

type Server struct {
	Url string
}

func New(root string) pkg.Gateway {
	configPath := filepath.Join(root, "proxy")
	// where should service-restart/node-reboot recovery be handled?
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err := os.MkdirAll(configPath, 0644)
		if err != nil {
			// ok to panic?
			panic(errors.Wrap(err, "couldn't make gateway config dir"))
		}
	}

	g := &gatewayModule{
		proxyConfigPath: configPath,
		traefikStarted:  false,
	}
	log.Debug().Bool("exist", namespace.Exists(types.PublicNamespace)).Msg("checking public namespace")
	if !isTraefikStarted() && namespace.Exists(types.PublicNamespace) {
		err := g.startTraefik()
		if err != nil {
			log.Error().Err(err).Msg("couldn't start traefik")
		}
	}
	return g
}
func isTraefikStarted() bool {
	z, err := zinit.New("")
	if err != nil {
		panic(errors.Wrap(err, "couldn't get zinit client"))
	}
	defer z.Close()
	started := true
	traefikStatus, err := z.Status(traefikService)
	if err != nil {
		started = false
	}
	log.Debug().Str("state", traefikStatus.State.String()).Msg("checking traefik state")
	started = traefikStatus.State.String() == zinit.ServiceStateRunning
	return started
}
func (g *gatewayModule) startTraefik() error {
	z, err := zinit.New("")
	if err != nil {
		return errors.Wrap(err, "couldn't get zinit client")
	}
	defer z.Close()
	cmd := fmt.Sprintf("ip netns exec public traefik --log.level=DEBUG --providers.file.directory=%s --providers.file.watch=true", g.proxyConfigPath)
	zinit.AddService(traefikService, zinit.InitService{
		Exec: cmd,
	})
	if err := z.Monitor(traefikService); err != nil {
		return errors.Wrap(err, "couldn't monitor traefik service")
	}
	if err := z.StartWait(time.Second*20, traefikService); err != nil {
		return errors.Wrap(err, "waiting for trafik start timed out")
	}
	g.traefikStarted = true
	return nil
}

func (g *gatewayModule) SetNamedProxy(fqdn string, backends []string) error {
	if !g.traefikStarted {
		if namespace.Exists(types.PublicNamespace) {
			if err := g.startTraefik(); err != nil {
				return errors.Wrap(err, "couldn't start traefik")
			}
		} else {
			return errors.New("node doesn't support gateway workloads as it doesn't have public config")
		}
	}

	rule := fmt.Sprintf("Host(`%s`) && PathPrefix(`/`)", fqdn)
	servers := make([]Server, len(backends))
	for idx, backend := range backends {
		servers[idx] = Server{
			Url: backend,
		}
	}
	config := ProxyConfig{
		Http: HTTPConfig{
			Routers: map[string]Router{
				fqdn: {
					Rule:    rule,
					Service: fqdn,
				},
			},
			Services: map[string]Service{
				fqdn: {
					LoadBalancer: LoadBalancer{
						Servers: servers,
					},
				},
			},
		},
	}

	yamlString, err := yaml.Marshal(&config)
	if err != nil {
		return errors.Wrap(err, "failed to convert config to yaml")
	}
	log.Debug().Str("yaml_config", string(yamlString)).Msg("configuration file")
	filename := filepath.Join(g.proxyConfigPath, fmt.Sprintf("%s.yaml", fqdn))
	if err = os.WriteFile(filename, yamlString, 0644); err != nil {
		return errors.Wrap(err, "couldn't open config file for writing")
	}

	return nil
}
func (g *gatewayModule) DeleteNamedProxy(fqdn string) error {
	filename := filepath.Join(g.proxyConfigPath, fmt.Sprintf("%s.yaml", fqdn))
	if err := os.Remove(filename); err != nil {
		return errors.Wrap(err, "couldn't remove config file")
	}
	return nil
}
