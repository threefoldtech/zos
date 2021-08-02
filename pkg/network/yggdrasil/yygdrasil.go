package yggdrasil

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
)

const (
	zinitService = "yggdrasil"
	confPath     = "/var/cache/modules/networkd/yggdrasil.conf"
)

// Server represent a yggdrasil server
type Server struct {
	zinit *zinit.Client
	cfg   *config.NodeConfig
}

// NewServer create a new yggdrasil Server
func NewServer(zinit *zinit.Client, cfg *config.NodeConfig) *Server {
	return &Server{
		zinit: zinit,
		cfg:   cfg,
	}
}

// Start creates an yggdrasil zinit service and starts it
func (s *Server) Start() error {
	status, err := s.zinit.Status(zinitService)
	if err == nil && status.State.Is(zinit.ServiceStateRunning) {
		return nil
	}

	if err := writeConfig(confPath, s.cfg); err != nil {
		return err
	}

	bin, err := exec.LookPath("yggdrasil")
	if err != nil {
		return err
	}

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: fmt.Sprintf("ip netns exec ndmz %s -useconffile %s -loglevel trace", bin, confPath),
		After: []string{
			"node-ready",
			"networkd",
		},
		Test: "yggdrasilctl getself | grep -i coords",
	})
	if err != nil {
		return err
	}

	if err := s.zinit.Monitor(zinitService); err != nil {
		return err
	}

	return s.zinit.StartWait(time.Second*20, zinitService)
}

// Stop stop the yggdrasil zinit service
func (s *Server) Stop() error {
	status, err := s.zinit.Status(zinitService)
	if err != nil {
		return err
	}

	if !status.State.Is(zinit.ServiceStateRunning) {
		return nil
	}

	return s.zinit.StopWait(time.Second*5, zinitService)
}

// NodeID returns the yggdrasil node ID of s
func (s *Server) NodeID() (ed25519.PublicKey, error) {
	if s.cfg.PublicKey == "" {
		panic("EncryptionPublicKey empty")
	}

	return hex.DecodeString(s.cfg.PublicKey)
}

// Address return the address in the 200::/7 subnet allocated by yggdrasil
func (s *Server) Address() (net.IP, error) {
	nodeID, err := s.NodeID()
	if err != nil {
		return nil, err
	}

	ip := make([]byte, net.IPv6len)
	copy(ip, address.AddrForKey(nodeID)[:])

	return ip, nil
}

// Subnet return the 300::/64 subnet allocated by yggdrasil
func (s *Server) Subnet() (net.IPNet, error) {
	nodeID, err := s.NodeID()
	if err != nil {
		return net.IPNet{}, err
	}

	snet := *address.SubnetForKey(nodeID)
	ipnet := net.IPNet{
		IP:   append(snet[:], 0, 0, 0, 0, 0, 0, 0, 0),
		Mask: net.CIDRMask(len(snet)*8, 128),
	}

	return ipnet, nil
}

// Gateway return the first IP of the 300::/64 subnet allocated by yggdrasil
func (s *Server) Gateway() (net.IPNet, error) {

	subnet, err := s.Subnet()
	if err != nil {
		return net.IPNet{}, err
	}
	subnet.IP[len(subnet.IP)-1] = 0x1

	return subnet, nil
}

// Tun return the name of the TUN interface created by yggdrasil
func (s *Server) Tun() string {
	return s.cfg.IfName
}

// SubnetFor return an IP address out of the node allocated subnet by hasing b and using it
// to generate the last 64 bits of the IPV6 address
func (s *Server) SubnetFor(b []byte) (net.IP, error) {
	subnet, err := s.Subnet()
	if err != nil {
		return nil, err
	}

	ip := make([]byte, net.IPv6len)
	copy(ip, subnet.IP)

	return subnetFor(ip, b)
}

func subnetFor(prefix net.IP, b []byte) (net.IP, error) {
	h := sha512.New()
	if _, err := h.Write(b); err != nil {
		return nil, err
	}
	digest := h.Sum(nil)
	copy(prefix[8:], digest[:8])
	return prefix, nil
}

func writeConfig(path string, cfg *config.NodeConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(cfg)
}
