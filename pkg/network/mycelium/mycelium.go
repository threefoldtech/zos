package mycelium

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
	"strings"
	"time"

	"github.com/oasisprotocol/curve25519-voi/primitives/x25519"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

const (
	MyIface      = "my0"
	tunName      = "utun9"
	zinitService = "mycelium"
	confPath     = "var/cache/modules/networkd/mycelium_priv_key.bin"
)

// MyServer represent a mycelium server
type MyServer struct {
	cfg *NodeConfig
}

// NewMyServer create a new mycelium Server
func NewMyServer(cfg *NodeConfig) *MyServer {
	return &MyServer{
		cfg: cfg,
	}
}

func (s *MyServer) Restart(z *zinit.Client) error {
	return z.Kill(zinitService, zinit.SIGTERM)
}

func (s *MyServer) Reload(z *zinit.Client) error {
	return z.Kill(zinitService, zinit.SIGHUP)
}

// Start creates a mycelium zinit service and starts it
func (s *MyServer) Ensure(z *zinit.Client, ns string) error {
	if !namespace.Exists(ns) {
		return fmt.Errorf("invalid namespace '%s'", ns)
	}

	if err := writeKey(confPath, s.cfg.PrivateKey); err != nil {
		return err
	}

	// service found.
	// better if we just stop, forget and start over to make
	// sure we using the right exec params
	if _, err := z.Status(zinitService); err == nil {
		// not here we need to stop it
		if err := z.StopWait(5*time.Second, zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to stop mycelium service")
		}
		if err := z.Forget(zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to forget mycelium service")
		}
	}

	bin, err := exec.LookPath("mycelium")
	if err != nil {
		return err
	}

	cmd := `sh -c '
    exec ip netns exec %s %s --key-file %s --tun-name %s --peers %s
    '`

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: fmt.Sprintf(cmd, ns, bin, confPath, s.cfg.TunName, strings.Join(s.cfg.Peers, " ")),
		After: []string{
			"node-ready",
		},
	})
	if err != nil {
		return err
	}

	if err := z.Monitor(zinitService); err != nil && !errors.Is(err, zinit.ErrAlreadyMonitored) {
		return err
	}

	return z.StartWait(time.Second*20, zinitService)
}

// -> this should be removed?
// NodeID returns mycelium node ID of s
func (s *MyServer) NodeID() (ed25519.PublicKey, error) {
	if s.cfg.PublicKey == "" {
		panic("EncryptionPublicKey empty")
	}

	return hex.DecodeString(s.cfg.PublicKey)
}

// -> how to get mycelium address
// Address return the address in the 400::/7 subnet allocated by mycelium
func (s *MyServer) Address() (net.IP, error) {
	nodeID, err := s.NodeID()
	if err != nil {
		return nil, err
	}

	ip := make([]byte, net.IPv6len)
	copy(ip, address.AddrForKey(nodeID)[:])

	return ip, nil
}

// -> how should I do this?
// Subnet return the 700::/64 subnet allocated by mycelium
func (s *MyServer) Subnet() (net.IPNet, error) {
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

// -> how should I do this?
// Gateway return the first IP of the 700::/64 subnet allocated by mycelium
func (s *MyServer) Gateway() (net.IPNet, error) {
	subnet, err := s.Subnet()
	if err != nil {
		return net.IPNet{}, err
	}
	subnet.IP[len(subnet.IP)-1] = 0x1

	return subnet, nil
}

// -> how should I do this?
// Tun return the name of the TUN interface created by mycelium
func (s *MyServer) Tun() string {
	return s.cfg.TunName
}

// -> how should I do this?
// SubnetFor return an IP address out of the node allocated subnet by hasing b and using it
// to generate the last 64 bits of the IPV6 address
func (s *MyServer) SubnetFor(b []byte) (net.IPNet, error) {
	subnet, err := s.Subnet()
	if err != nil {
		return net.IPNet{}, err
	}

	ip := make([]byte, net.IPv6len)
	copy(ip, subnet.IP)

	subIP, err := subnetFor(ip, b)
	if err != nil {
		return net.IPNet{}, err
	}

	return net.IPNet{
		IP:   subIP,
		Mask: net.CIDRMask(64, 128),
	}, nil
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

func writeKey(path string, privateKey ed25519.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	key := x25519.EdPrivateKeyToX25519([]byte(privateKey))

	return json.NewEncoder(f).Encode(key)
}
