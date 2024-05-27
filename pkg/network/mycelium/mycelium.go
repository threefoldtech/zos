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
)

const (
	MyIface      = "my0"
	tunName      = "utun9"
	myBin        = "mycelium"
	zinitService = "mycelium"
	MyListenTCP  = 9651
	confPath     = "/tmp/mycelium_priv_key.bin"
)

// MyServer represent a mycelium server
type MyServer struct {
	cfg *NodeConfig
	ns  string
}

type MyceliumInspection struct {
	PublicKey string `json:"publicKey"`
	Address   net.IP `json:"address"`
}

type Subnet [8]byte

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
	s.ns = ns

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

	bin, err := exec.LookPath(myBin)
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

// NodeID returns mycelium node ID of s
func (s *MyServer) NodeID() (ed25519.PublicKey, error) {
	if s.cfg.PublicKey == "" {
		panic("EncryptionPublicKey empty")
	}

	return hex.DecodeString(s.cfg.PublicKey)
}

func (s *MyServer) InspectMycelium() (inspection MyceliumInspection, err error) {
	// we check if the file exists before we do inspect because mycelium will create a random seed
	// file if file does not exist
	_, err = os.Stat(confPath)
	if err != nil {
		return inspection, err
	}

	bin, err := exec.LookPath(myBin)
	if err != nil {
		return inspection, err
	}

	cmd := fmt.Sprintf(`exec ip netns exec %s %s --key-file %s inspect --json`, s.ns, bin, confPath)
	output, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return inspection, errors.Wrap(err, "failed to inspect mycelium ip")
	}

	if err := json.Unmarshal(output, &inspection); err != nil {
		return inspection, errors.Wrap(err, "failed to load mycelium information from key")
	}

	return
}

// IP return the address in the 400::/7 subnet allocated by mycelium
func (m *MyceliumInspection) IP() net.IP {
	return net.IP(m.Address)
}

// Subnet return the 400::/64 subnet allocated by mycelium
func (m *MyceliumInspection) Subnet() (subnet net.IPNet, err error) {
	ipv6 := m.Address.To16()
	if ipv6 == nil {
		return subnet, errors.Errorf("invalid mycelium ip")
	}

	ip := make(net.IP, net.IPv6len)
	copy(ip[0:8], ipv6[0:8])

	subnet = net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(64, 128),
	}

	return
}

// Gateway derive the gateway IP from the mycelium IP in the /64 range.
func (m *MyceliumInspection) Gateway() (gw net.IPNet, err error) {
	subnet, err := m.Subnet()
	if err != nil {
		return gw, err
	}

	ip := subnet.IP
	ip[net.IPv6len-1] = 1

	gw = net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(64, 128),
	}

	return
}

// Tun return the name of the TUN interface created by mycelium
func (s *MyServer) Tun() string {
	return s.cfg.TunName
}

// SubnetFor return an IP address out of the node allocated subnet by hasing b and using it
// to generate the last 64 bits of the IPV6 address
func (m *MyceliumInspection) SubnetFor(b []byte) (net.IPNet, error) {
	subnet, err := m.Subnet()
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
