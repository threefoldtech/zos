package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/provision"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/emicklei/dot"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

func cmdGraphNetwork(c *cli.Context) error {
	var (
		network = &pkg.Network{}
		schema  = c.GlobalString("schema")
		err     error
	)

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	outfile, err := os.Create(schema + ".dot")
	if err != nil {
		return err
	}

	return networkGraph(*network, outfile)
}

func cmdCreateNetwork(c *cli.Context) error {
	name := c.String("name")
	if name == "" {
		return fmt.Errorf("network name cannot be empty")
	}
	ipRange := c.String("cidr")
	if ipRange == "" {
		return fmt.Errorf("ip range cannot be empty")
	}

	ipnet, err := types.ParseIPNet(ipRange)
	if err != nil {
		errors.Wrap(err, "invalid ip range")
	}
	network := &pkg.Network{
		Name:         name,
		IPRange:      ipnet,
		NetResources: []pkg.NetResource{},
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("schema"), r)
}

func cmdsAddNode(c *cli.Context) error {
	var (
		network = &pkg.Network{}
		schema  = c.GlobalString("schema")
		err     error

		nodeID = c.String("node")
		subnet = c.String("subnet")
		port   = c.Uint("port")

		forceHidden = c.Bool("force-hidden")
	)

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	if subnet == "" {
		return fmt.Errorf("subnet cannot be empty")
	}
	ipnet, err := types.ParseIPNet(subnet)
	if err != nil {
		return errors.Wrap(err, "invalid subnet")
	}

	if port == 0 {
		port, err = pickPort(nodeID)
		if err != nil {
			return errors.Wrap(err, "failed to pick wireguard port")
		}
	}

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return errors.Wrap(err, "error during wireguard key generation")
	}
	sk := privateKey.String()

	pk, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		return errors.Wrap(err, "failed to parse nodeID")
	}

	encrypted, err := crypto.Encrypt([]byte(sk), pk)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt private key")
	}

	var endpoints []net.IP
	if !forceHidden {
		pubSubnets, err := getEndPointAddrs(pkg.StrIdentifier(nodeID))
		if err != nil {
			return errors.Wrap(err, "failed to get node public endpoints")
		}

		// In rust this bullshit could be written easily as:
		// let ips = pubEndpoints.into_iter().map(|n| n.IP).collect();
		// but alas
		for _, sn := range pubSubnets {
			endpoints = append(endpoints, sn.IP)
		}
	}

	nr := pkg.NetResource{
		NodeID:       nodeID,
		PubEndpoints: endpoints,
		Subnet:       ipnet,
		WGListenPort: uint16(port),
		WGPublicKey:  privateKey.PublicKey().String(),
		WGPrivateKey: hex.EncodeToString(encrypted),
	}

	network.NetResources = append(network.NetResources, nr)

	if err = generatePeers(network); err != nil {
		return errors.Wrap(err, "failed to generate peers")
	}

	r, err := embed(network, provision.NetworkReservation)
	if err != nil {
		return err
	}

	return output(schema, r)
}

func cmdsRemoveNode(c *cli.Context) error {
	var (
		network = &pkg.Network{}
		schema  = c.GlobalString("schema")
		nodeID  = c.String("node")
		err     error
	)

	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	network, err = loadNetwork(schema)
	if err != nil {
		return err
	}

	for i, nr := range network.NetResources {
		if nr.NodeID == nodeID {
			network.NetResources = append(network.NetResources[:i], network.NetResources[i+1:]...)
			break
		}
	}

	// we don't remove the peer from the other network resource
	// while this is dirty wireguard doesn't really care
	raw, err := json.Marshal(network)
	if err != nil {
		return err
	}

	r := &provision.Reservation{
		Type: provision.NetworkReservation,
		Data: raw,
	}
	// r, err := embed(network, provision.NetworkReservation)
	// if err != nil {
	//	return err
	// }

	return output(schema, r)
}

func loadNetwork(name string) (network *pkg.Network, err error) {
	network = &pkg.Network{}

	if name == "" {
		return nil, fmt.Errorf("schema name cannot be empty")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := &provision.Reservation{}
	if err := json.NewDecoder(f).Decode(r); err != nil {
		return nil, errors.Wrapf(err, "failed to decode json encoded reservation at %s", name)
	}

	if err := json.Unmarshal(r.Data, network); err != nil {
		return nil, errors.Wrapf(err, "failed to decode json encoded network at %s", name)
	}
	return network, nil
}

func pickPort(nodeID string) (uint, error) {
	node, err := client.GetNode(pkg.StrIdentifier(nodeID))
	if err != nil {
		return 0, err
	}

	p := uint(rand.Intn(6000) + 2000)

	for isIn(node.WGPorts, p) {
		p = uint(rand.Intn(6000) + 2000)
	}
	return p, nil
}

// a node has either a public namespace with []ipv4 or/and []ipv6 -or-
// some interface has received a SLAAC addr
// which has been registered in BCDB
func getEndPointAddrs(nodeID pkg.Identifier) ([]types.IPNet, error) {
	node, err := client.GetNode(nodeID)
	if err != nil {
		return nil, err
	}
	var endpoints []types.IPNet
	if node.PublicConfig != nil {
		if node.PublicConfig.IPv4.IP != nil {
			ip := node.PublicConfig.IPv4.IP
			if ip.IsGlobalUnicast() && !isPrivateIP(ip) {
				endpoints = append(endpoints, node.PublicConfig.IPv4)
			}
		}
		if node.PublicConfig.IPv6.IP != nil {
			ip := node.PublicConfig.IPv6.IP
			if ip.IsGlobalUnicast() && !isPrivateIP(ip) {
				endpoints = append(endpoints, node.PublicConfig.IPv6)
			}
		}
	} else {
		for _, iface := range node.Ifaces {
			for _, ip := range iface.Addrs {
				if !ip.IP.IsGlobalUnicast() || isPrivateIP(ip.IP) {
					continue
				}
				endpoints = append(endpoints, ip)
			}
		}
	}
	// If the length is 0, then its a hidden node
	return endpoints, nil
}

func isIn(l []uint, i uint) bool {
	for _, x := range l {
		if i == x {
			return true
		}
	}
	return false
}

func hasIPv4(n pkg.NetResource) bool {
	for _, pep := range n.PubEndpoints {
		if pep.To4() != nil {
			return true
		}
	}
	return false
}

// This function assumes:
// - that a hidden node has functioning IPv4
// - that a public node ALWAYS has public IPv6, and OPTIONALLY public IPv4
// - that any public endpoint on any node is actually reachable (i.e. no firewall
//		blocking incomming traffic)
func generatePeers(n *pkg.Network) error {
	// Find public node, which will be used to connect all hidden nodes.
	// In case there are hidden nodes, the public node needs IPv4 support as well.
	var hasHiddenNodes bool
	for _, nr := range n.NetResources {
		if len(nr.PubEndpoints) == 0 {
			hasHiddenNodes = true
			break
		}
	}

	// Look for a public node to connect hidden nodes. This is only needed
	// in case there are hidden nodes.
	var pubNr string
	if hasHiddenNodes {
		for _, nr := range n.NetResources {
			if hasIPv4(nr) {
				pubNr = nr.NodeID
				break
			}
		}
		if pubNr == "" {
			return errors.New("Network has hidden nodes but no public IPv4 node exists")
		}
	}

	// Find all hidden nodes, and collect their subnets. Also collect the subnets
	// of public IPv6 only nodes, since hidden nodes need IPv4 to connect.
	hiddenSubnets := make(map[string]types.IPNet)
	// also maintain subnets from nodes who have only IPv6 since this will also
	// need to be routed for hidden nodes
	ipv6OnlySubnets := make(map[string]types.IPNet)
	for _, nr := range n.NetResources {
		if len(nr.PubEndpoints) == 0 {
			hiddenSubnets[nr.NodeID] = nr.Subnet
			continue
		}
		if !hasIPv4(nr) {
			ipv6OnlySubnets[nr.NodeID] = nr.Subnet
		}
	}

	for i := range n.NetResources {
		// Note: we need to loop by index and manually asign nr, doing
		// for _, nr := range ... causes nr to be copied, meaning we can't modify
		// it in place
		nr := &n.NetResources[i]
		nr.Peers = []pkg.Peer{}
		for _, onr := range n.NetResources {
			if nr.NodeID == onr.NodeID {
				continue
			}

			if len(nr.PubEndpoints) == 0 {
				// If node is hidden, set only public peers (with IPv4), and set first public peer to
				// contain all hidden subnets, except for the one owned by the node
				if !hasIPv4(onr) {
					continue
				}

				allowedIPs := make([]types.IPNet, 2)
				allowedIPs[0] = onr.Subnet
				allowedIPs[1] = types.NewIPNet(wgIP(&onr.Subnet.IPNet))

				// Also add all other subnets if this is the pub node
				if onr.NodeID == pubNr {
					for owner, subnet := range hiddenSubnets {
						// Do not add our own subnet
						if owner == nr.NodeID {
							continue
						}

						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, types.NewIPNet(wgIP(&subnet.IPNet)))
					}

					for _, subnet := range ipv6OnlySubnets {
						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, types.NewIPNet(wgIP(&subnet.IPNet)))
					}
				}

				// Endpoint must be IPv4
				var endpoint string
				for _, pep := range onr.PubEndpoints {
					if pep.To4() != nil {
						endpoint = fmt.Sprintf("%s:%d", pep.String(), nr.WGListenPort)
					}
				}

				nr.Peers = append(nr.Peers, pkg.Peer{
					WGPublicKey: onr.WGPublicKey,
					Subnet:      onr.Subnet,
					AllowedIPs:  allowedIPs,
					Endpoint:    endpoint,
				})
			} else {
				// if we are not hidden, we add all other nodes, unless we don't
				// have IPv4, because then we also can't connect to hidden nodes.
				// Ignore hidden nodes if we don't have IPv4
				if !hasIPv4(*nr) && len(onr.PubEndpoints) == 0 {
					continue
				}

				allowedIPs := make([]types.IPNet, 2)
				allowedIPs[0] = onr.Subnet
				allowedIPs[1] = types.NewIPNet(wgIP(&onr.Subnet.IPNet))

				// if the peer is hidden but we have IPv4,  we can connect to it, but we don't know
				// an endpoint.
				if len(onr.PubEndpoints) == 0 {
					nr.Peers = append(nr.Peers, pkg.Peer{
						WGPublicKey: onr.WGPublicKey,
						Subnet:      onr.Subnet,
						AllowedIPs:  allowedIPs,
						Endpoint:    "",
					})
					continue
				}

				// both nodes are public therefore we can connect over IPv6

				// if this is the selected pubNr - also need to add allowedIPs
				// for the hidden nodes
				if onr.NodeID == pubNr {
					for _, subnet := range hiddenSubnets {
						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, types.NewIPNet(wgIP(&subnet.IPNet)))
					}
				}

				var endpoint string
				for _, pep := range onr.PubEndpoints {
					if pep.To16() != nil {
						endpoint = fmt.Sprintf("[%s]:%d", pep.String(), onr.WGListenPort)
					}
				}

				nr.Peers = append(nr.Peers, pkg.Peer{
					WGPublicKey: onr.WGPublicKey,
					Subnet:      onr.Subnet,
					AllowedIPs:  allowedIPs,
					Endpoint:    endpoint,
				})

			}
		}

	}

	return nil
}

func networkGraph(n pkg.Network, w io.Writer) error {
	nodes := make(map[string]dot.Node)
	graph := dot.NewGraph(dot.Directed)

	for _, nr := range n.NetResources {
		node := graph.Node(strings.Join([]string{nr.NodeID, nr.Subnet.String()}, "\n")).Box()
		// set special style for "hidden" nodes
		if len(nr.PubEndpoints) == 0 {
			node.Attr("style", "dashed")
			node.Attr("color", "blue")
		}
		nodes[nr.WGPublicKey] = node
	}

	for _, nr := range n.NetResources {
		for _, peer := range nr.Peers {
			allowedIPs := make([]string, 0, len(peer.AllowedIPs)/2)
			for _, aip := range peer.AllowedIPs {
				if !isCGN(aip) {
					allowedIPs = append(allowedIPs, aip.String())
				}
			}

			edge := graph.Edge(nodes[nr.WGPublicKey], nodes[peer.WGPublicKey], strings.Join(allowedIPs, "\n"))
			if peer.Endpoint == "" {
				// connections to this peer are IPv4 -> blue, and can not be initiated by this node -> dashed
				edge.Attr("color", "blue").Attr("style", "dashed")
				continue
			}
			if net.ParseIP(peer.Endpoint[:strings.LastIndex(peer.Endpoint, ":")]).To4() != nil {
				// IPv4 connection -> blue
				edge.Attr("color", "blue")
			}
		}
	}

	graph.Write(w)
	return nil
}

func wgIP(subnet *net.IPNet) *net.IPNet {
	// example: 10.3.1.0 -> 100.64.3.1
	a := subnet.IP[len(subnet.IP)-3]
	b := subnet.IP[len(subnet.IP)-2]

	return &net.IPNet{
		IP:   net.IPv4(0x64, 0x40, a, b),
		Mask: net.CIDRMask(32, 32),
	}
}

func isPrivateIP(ip net.IP) bool {
	privateIPBlocks := []*net.IPNet{}
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func isCGN(subnet types.IPNet) bool {
	_, block, err := net.ParseCIDR("100.64.0.0/10")
	if err != nil {
		panic(err)
	}
	return block.Contains(subnet.IP)
}
