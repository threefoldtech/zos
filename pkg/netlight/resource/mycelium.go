package resource

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos4/pkg/netlight/resource/peers"
	"github.com/threefoldtech/zos4/pkg/zinit"
)

const (
	myceliumBin     = "mycelium"
	MyceliumSeedDir = "/tmp/network/mycelium"

	myceliumSeedLen = 6
)

var (
	myceliumIpBase = []byte{
		0xff, 0x0f,
	}

	invalidMyceliumSeeds = [][]byte{
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}
)

type MyceliumInspection struct {
	PublicKey string `json:"publicKey"`
	Address   net.IP `json:"address"`
}

func InspectMycelium(seed []byte) (inspection MyceliumInspection, err error) {
	// we check if the file exists before we do inspect because mycelium will create a random seed
	// file if file does not exist
	tmp, err := os.CreateTemp("", "my-inspect")
	if err != nil {
		return inspection, err
	}

	defer os.RemoveAll(tmp.Name())

	if _, err := tmp.Write(seed); err != nil {
		return inspection, fmt.Errorf("failed to write seed: %w", err)
	}

	tmp.Close()

	return inspectMyceliumFile(tmp.Name())
}

func inspectMyceliumFile(path string) (inspection MyceliumInspection, err error) {
	output, err := exec.Command(myceliumBin, "inspect", "--json", "--key-file", path).Output()
	if err != nil {
		return inspection, fmt.Errorf("failed to inspect mycelium ip: %w", err)
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

func (m *MyceliumInspection) IPFor(seed []byte) (ip net.IPNet, gw net.IPNet, err error) {
	if len(seed) != myceliumSeedLen {
		return ip, gw, fmt.Errorf("invalid seed length")
	}

	if slices.ContainsFunc(invalidMyceliumSeeds, func(b []byte) bool {
		return slices.Equal(seed, b)
	}) {
		return ip, gw, fmt.Errorf("invalid seed")
	}

	// first find the base subnet.
	gw, err = m.Gateway()
	if err != nil {
		return ip, gw, err
	}

	ip = net.IPNet{
		IP:   slices.Clone(gw.IP),
		Mask: slices.Clone(gw.Mask),
	}

	// the subnet already have the /64 part of the network (that's 8 bytes)
	// we then add a fixed 2 bytes this will avoid reusing the same gw or
	// the device ip
	copy(ip.IP[8:10], myceliumIpBase)
	// then finally we use the 6 bytes seed to build the rest of the IP
	copy(ip.IP[10:16], seed)

	return
}

func setupMycelium(netNS ns.NetNS, mycelium string, seed []byte) error {
	if err := os.MkdirAll(MyceliumSeedDir, 0744); err != nil {
		return fmt.Errorf("failed to create seed temp location: %w", err)
	}

	inspect, err := InspectMycelium(seed)
	if err != nil {
		return err
	}

	gw, err := inspect.Gateway()
	if err != nil {
		return err
	}

	err = netNS.Do(func(nn ns.NetNS) error {
		return setLinkAddr(mycelium, &gw)
	})

	if err != nil {
		return err
	}

	// - fetch peers
	// - write seed file
	// - create zinit config
	// - monitor

	list, err := peers.PeersList()
	if err != nil {
		return err
	}

	name := filepath.Base(netNS.Path())
	if err := os.WriteFile(filepath.Join(MyceliumSeedDir, name), seed, 0444); err != nil {
		return fmt.Errorf("failed to create seed file '%s': %w", name, err)
	}

	return ensureMyceliumService(zinit.Default(), name, list)
}

// Start creates a mycelium zinit service and starts it
func ensureMyceliumService(z *zinit.Client, namespace string, peers []string) error {
	zinitService := fmt.Sprintf("mycelium-%s", namespace)
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

	bin, err := exec.LookPath(myceliumBin)
	if err != nil {
		return err
	}

	args := []string{
		"ip", "netns", "exec", namespace,
		bin,
		"--silent",
		"--key-file", filepath.Join(MyceliumSeedDir, namespace),
		"--tun-name", "my0",
		"--peers",
	}

	args = append(args, peers...)

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: strings.Join(args, " "),
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

func destroyMycelium(netNS ns.NetNS, z *zinit.Client) error {
	name := filepath.Base(netNS.Path())

	zinitService := fmt.Sprintf("mycelium-%s", name)

	if err := z.StopWait(5*time.Second, zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
		return fmt.Errorf("failed to stop service %q: %w", zinitService, err)
	}
	if err := z.Forget(zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
		return fmt.Errorf("failed to forget service %q: %w", zinitService, err)
	}

	return os.Remove(filepath.Join(MyceliumSeedDir, name))
}
