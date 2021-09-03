package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
	"gopkg.in/yaml.v2"
)

const (
	traefikService = "traefik"
)

type gatewayModule struct {
	networker *stubs.NetworkerStub

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

func New(ctx context.Context, networker *stubs.NetworkerStub, root string) (pkg.Gateway, error) {
	configPath := filepath.Join(root, "proxy")
	// where should service-restart/node-reboot recovery be handled?
	err := os.MkdirAll(configPath, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make gateway config dir")
	}

	g := &gatewayModule{
		networker:       networker,
		proxyConfigPath: configPath,
		traefikStarted:  false,
	}
	traefikStarted, err := isTraefikStarted()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't check traefik status")
	}
	supported, err := g.isGatewaySupported(ctx)
	if err != nil {
		supported = false
		log.Warn().Err(err).Msg("failed to get public config")
	}
	if !traefikStarted && supported {
		err := g.startTraefik()
		if err != nil {
			log.Error().Err(err).Msg("couldn't start traefik")
		}
	}
	return g, nil
}

func isTraefikStarted() (bool, error) {
	z, err := zinit.New("")
	if err != nil {
		return false, errors.Wrap(err, "couldn't get zinit client")
	}
	defer z.Close()
	started := true
	traefikStatus, err := z.Status(traefikService)
	if err != nil {
		started = false
	}
	log.Debug().Str("state", traefikStatus.State.String()).Msg("checking traefik state")
	started = traefikStatus.State.Is(zinit.ServiceStateRunning)
	return started, nil
}

func (g *gatewayModule) isGatewaySupported(ctx context.Context) (bool, error) {
	cfg, err := g.networker.GetPublicConfig(ctx)
	if err != nil {
		return false, errors.Wrap(err, "couldn't get public config")
	}
	cfg.Domain = "omar.com"
	return cfg.Domain != "", err
}

func (g *gatewayModule) getDomainName(ctx context.Context) (string, error) {
	cfg, err := g.networker.GetPublicConfig(ctx)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get public config")
	}
	cfg.Domain = "omar.com"
	return cfg.Domain, err
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

func (g *gatewayModule) SetNamedProxy(wlID string, prefix string, backends []string) (string, error) {
	ctx := context.TODO()
	domain, err := g.getDomainName(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get node's domain")
	} else if domain == "" {
		return "", errors.New("node doesn't support gateway workloads")
	}
	fqdn := fmt.Sprintf("%s.%s", prefix, domain)

	if !g.traefikStarted {
		if err := g.startTraefik(); err != nil {
			return "", errors.Wrap(err, "couldn't start traefik")
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
				wlID: {
					Rule:    rule,
					Service: wlID,
				},
			},
			Services: map[string]Service{
				wlID: {
					LoadBalancer: LoadBalancer{
						Servers: servers,
					},
				},
			},
		},
	}

	yamlString, err := yaml.Marshal(&config)
	if err != nil {
		return "", errors.Wrap(err, "failed to convert config to yaml")
	}
	log.Debug().Str("yaml_config", string(yamlString)).Msg("configuration file")
	filename := filepath.Join(g.proxyConfigPath, fmt.Sprintf("%s.yaml", wlID))
	if err = os.WriteFile(filename, yamlString, 0644); err != nil {
		return "", errors.Wrap(err, "couldn't open config file for writing")
	}

	return fqdn, nil
}
func (g *gatewayModule) DeleteNamedProxy(wlID string) error {
	filename := filepath.Join(g.proxyConfigPath, fmt.Sprintf("%s.yaml", wlID))
	if err := os.Remove(filename); err != nil {
		return errors.Wrap(err, "couldn't remove config file")
	}
	return nil
}
