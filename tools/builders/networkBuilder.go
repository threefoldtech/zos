package builders

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"

	"github.com/emicklei/dot"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/client"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// NetworkBuilder is a struct that can build networks
type NetworkBuilder struct {
	workloads.Network

	NodeID string
	Bcdb   *client.Client
	Mainui *identity.UserIdentity

	AccessPoints []AccessPoint `json:"access_points,omitempty"`

	// NetResources field override
	NetResources []NetResource `json:"net_resources"`
}

// NetResource is the description of a part of a network local to a specific node
type NetResource struct {
	workloads.NetworkNetResource

	// Public endpoints
	PubEndpoints []net.IP `json:"pub_endpoints"`
}

// AccessPoint info for a network, defining a node which will act as the AP, and
// the subnet which will be routed through it
type AccessPoint struct {
	// NodeID of the access point in the network
	NodeID string `json:"node_id"`
	// Subnet to be routed through this access point
	Subnet      schema.IPRange `json:"subnet"`
	WGPublicKey string         `json:"wg_public_key"`
	IP4         bool           `json:"ip4"`
}

// NewNetworkBuilder creates a new network builder
func NewNetworkBuilder(name string) *NetworkBuilder {
	return &NetworkBuilder{
		Network: workloads.Network{
			Name: name,
		},
	}
}

// LoadNetworkBuilder loads a network builder based on a file path
func LoadNetworkBuilder(reader io.Reader) (*NetworkBuilder, error) {
	network := workloads.Network{}

	err := json.NewDecoder(reader).Decode(&network)
	if err != nil {
		return &NetworkBuilder{}, err
	}

	return &NetworkBuilder{Network: network}, nil
}

// Save saves the network builder to an IO.Writer
func (n *NetworkBuilder) Save(writer io.Writer) error {
	err := json.NewEncoder(writer).Encode(n.Network)
	if err != nil {
		return err
	}
	return err
}

// Build returns the network
func (n *NetworkBuilder) Build() workloads.Network {
	return n.Network
}

// TODO ADD NODE ID TO NETWORK?

// WithName sets the ip range to the network
func (n *NetworkBuilder) WithName(name string) *NetworkBuilder {
	n.Network.Name = name
	return n
}

// WithIPRange sets the ip range to the network
func (n *NetworkBuilder) WithIPRange(ipRange schema.IPRange) *NetworkBuilder {
	n.Network.Iprange = ipRange
	return n
}

// WithStatsAggregator sets the stats aggregators to the network
func (n *NetworkBuilder) WithStatsAggregator(aggregators []workloads.StatsAggregator) *NetworkBuilder {
	n.Network.StatsAggregator = aggregators
	return n
}

// WithNetworkResources sets the network resources to the network
func (n *NetworkBuilder) WithNetworkResources(netResources []workloads.NetworkNetResource) *NetworkBuilder {
	n.Network.NetworkResources = netResources
	return n
}

// AddNode adds a node
func (n *NetworkBuilder) AddNode(networkSchema string, nodeID string, subnet string, port uint, forceHidden bool) error {
	n.NodeID = nodeID

	if subnet == "" {
		return fmt.Errorf("subnet cannot be empty")
	}
	ipnet, err := types.ParseIPNet(subnet)
	if err != nil {
		return errors.Wrap(err, "invalid subnet")
	}

	if port == 0 {
		port, err = n.pickPort()
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

	pubSubnets, err := n.getEndPointAddrs(pkg.StrIdentifier(nodeID))
	if err != nil {
		return errors.Wrap(err, "failed to get node public endpoints")
	}
	var endpoints []net.IP
	if !forceHidden {
		for _, sn := range pubSubnets {
			endpoints = append(endpoints, sn.IP)
		}
	}

	nr := NetResource{
		NetworkNetResource: workloads.NetworkNetResource{
			NodeId:                       nodeID,
			Iprange:                      schema.IPRange{ipnet.IPNet},
			WireguardListenPort:          int64(port),
			WireguardPublicKey:           privateKey.PublicKey().String(),
			WireguardPrivateKeyEncrypted: hex.EncodeToString(encrypted),
		},
		PubEndpoints: endpoints,
	}

	n.NetResources = append(n.NetResources, nr)

	if err = n.generatePeers(); err != nil {
		return errors.Wrap(err, "failed to generate peers")
	}

	f, err := os.Open(networkSchema)
	if err != nil {
		return errors.Wrap(err, "failed to open network schema")
	}
	return n.Save(f)
}

// AddAccess adds access to a network node
func (n *NetworkBuilder) AddAccess(networkSchema string, nodeID string, subnet string, wgPubKey string, ip4 bool) (string, error) {
	if nodeID == "" {
		return "", fmt.Errorf("nodeID cannot be empty")
	}

	if subnet == "" {
		return "", fmt.Errorf("subnet cannot be empty")
	}
	ipnet, err := types.ParseIPNet(subnet)
	if err != nil {
		return "", errors.Wrap(err, "invalid subnet")
	}

	var nodeExists bool
	var node NetResource
	for _, nr := range n.NetResources {
		if nr.NodeId == nodeID {
			node = nr
			nodeExists = true
			break
		}
	}

	if !nodeExists {
		return "", errors.New("can not add access through a node which is not in the network")
	}

	if len(node.PubEndpoints) == 0 {
		return "", errors.New("access node must have at least 1 public endpoint")
	}

	var endpoint string
	for _, ep := range node.PubEndpoints {
		if ep.To4() != nil {
			// ipv4 address
			if ip4 {
				endpoint = fmt.Sprintf("%s:%d", ep.String(), node.WireguardListenPort)
				break
			}
			// we want ipv6 so use the next address
			continue
		}
		if ep.To16() != nil {
			// due to the previous branch this can now only be an ipv6 address
			if !ip4 {
				endpoint = fmt.Sprintf("[%s]:%d", node.PubEndpoints[0].String(), node.WireguardListenPort)
				break
			}
			// we want ipv4 so use next address
			continue
		}
	}
	if endpoint == "" {
		return "", errors.New("access node has no public endpoint of the requested type")
	}

	var privateKey wgtypes.Key
	if wgPubKey == "" {
		privateKey, err = wgtypes.GeneratePrivateKey()
		if err != nil {
			return "", errors.Wrap(err, "error during wireguard key generation")
		}
		wgPubKey = privateKey.PublicKey().String()
	}

	ap := AccessPoint{
		NodeID:      nodeID,
		Subnet:      schema.IPRange{ipnet.IPNet},
		WGPublicKey: wgPubKey,
		IP4:         ip4,
	}

	n.AccessPoints = append(n.AccessPoints, ap)

	if err = n.generatePeers(); err != nil {
		return "", errors.Wrap(err, "failed to generate peers")
	}

	wgConf, err := genWGQuick(privateKey.String(), ipnet, node.WireguardPublicKey, n.Network.Iprange, endpoint)
	if err != nil {
		return "", err
	}

	f, err := os.Open(networkSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to open network schema")
	}
	return wgConf, n.Save(f)
}

// RemoveNode removes a node
func (n *NetworkBuilder) RemoveNode(schema string, nodeID string) error {
	for i, nr := range n.NetResources {
		if nr.NodeId == nodeID {
			n.NetResources = append(n.NetResources[:i], n.NetResources[i+1:]...)
			break
		}
	}

	f, err := os.Open(schema)
	if err != nil {
		return errors.Wrap(err, "failed to open network schema")
	}
	return n.Save(f)
}

func (n *NetworkBuilder) setPubEndpoints() error {
	for i := range n.NetResources {
		pep, err := n.getEndPointAddrs(pkg.StrIdentifier(n.NetResources[i].NodeId))
		if err != nil {
			return err
		}

		var endpoints []net.IP
		for _, sn := range pep {
			endpoints = append(endpoints, sn.IP)
		}

		n.NetResources[i].PubEndpoints = endpoints
	}

	// remove the pub endpoints from nodes which we assume have been marked
	// as force hidden
	hiddenNodes := make(map[string]struct{})
	for _, nr := range n.NetResources {
		if len(nr.PubEndpoints) > 0 {
			for _, peer := range nr.Peers {
				if peer.Endpoint == "" {
					hiddenNodes[peer.PublicKey] = struct{}{}
				}
			}
		}
	}

	for i := range n.NetResources {
		if _, exists := hiddenNodes[n.NetResources[i].WireguardPublicKey]; exists {
			n.NetResources[i].PubEndpoints = nil
		}
	}

	return nil
}

// LoadNetwork loads the network builder and does some extra operations first
func LoadNetwork(name string) (*NetworkBuilder, error) {
	if name == "" {
		return nil, fmt.Errorf("schema name cannot be empty")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	networkBuilder, err := LoadNetworkBuilder(f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load network builder")
	}

	if err = networkBuilder.setPubEndpoints(); err != nil {
		return nil, err
	}

	networkBuilder.extractAccessPoints()

	return networkBuilder, nil
}

func (n *NetworkBuilder) pickPort() (uint, error) {
	node, err := n.Bcdb.Directory.NodeGet(n.NodeID, false)
	if err != nil {
		return 0, err
	}

	p := uint(rand.Intn(6000) + 2000)

	for isIn(node.WgPorts, p) {
		p = uint(rand.Intn(6000) + 2000)
	}
	return p, nil
}

func isIn(l []int64, i uint) bool {
	for _, x := range l {
		if int64(i) == x {
			return true
		}
	}
	return false
}

func hasIPv4(n NetResource) bool {
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
//		blocking incoming traffic)
func (n *NetworkBuilder) generatePeers() error {
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
				pubNr = n.NodeID
				break
			}
		}
		if pubNr == "" {
			return errors.New("Network has hidden nodes but no public IPv4 node exists")
		}
	}

	// We also need to inform nodes how to route the external access subnets.
	// Working with the knowledge that these external subnets come in through
	// the network through a single access point, which is part of the network
	// and thus already routed, we can map the external subnets to the subnet
	// of the access point, and add these external subnets to all peers who also
	// have the associated internal subnet.
	//
	// Map the network subnets to their respective node ids first for easy access later
	internalSubnets := make(map[string]schema.IPRange)
	for _, nr := range n.NetResources {
		internalSubnets[n.NodeID] = nr.Iprange
	}

	externalSubnets := make(map[string][]schema.IPRange) // go does not like `types.IPNet` as key
	for _, ap := range n.AccessPoints {
		externalSubnets[internalSubnets[ap.NodeID].String()] = append(externalSubnets[internalSubnets[ap.NodeID].String()], ap.Subnet)
	}

	// Maintain a mapping of access point nodes to the subnet and wg key they give access
	// to, as these need to be added as peers as well for these nodes
	accessPoints := make(map[string][]AccessPoint)
	for _, ap := range n.AccessPoints {
		accessPoints[ap.NodeID] = append(accessPoints[ap.NodeID], ap)
	}

	// Find all hidden nodes, and collect their subnets. Also collect the subnets
	// of public IPv6 only nodes, since hidden nodes need IPv4 to connect.
	hiddenSubnets := make(map[string]schema.IPRange)
	// also maintain subnets from nodes who have only IPv6 since this will also
	// need to be routed for hidden nodes
	ipv6OnlySubnets := make(map[string]schema.IPRange)
	for _, nr := range n.NetResources {
		if len(nr.PubEndpoints) == 0 {
			hiddenSubnets[n.NodeID] = nr.Iprange
			continue
		}
		if !hasIPv4(nr) {
			ipv6OnlySubnets[n.NodeID] = nr.Iprange
		}
	}

	for i := range n.NetResources {
		// Note: we need to loop by index and manually assign nr, doing
		// for _, nr := range ... causes nr to be copied, meaning we can't modify
		// it in place
		nr := &n.NetResources[i]
		nr.Peers = []workloads.WireguardPeer{}
		for _, onr := range n.NetResources {
			if n.NodeID == onr.NodeId {
				continue
			}

			allowedIPs := make([]schema.IPRange, 2)
			allowedIPs[0] = onr.Iprange
			allowedIPs[1] = *wgIP(&onr.Iprange)

			var endpoint string

			if len(nr.PubEndpoints) == 0 {
				// If node is hidden, set only public peers (with IPv4), and set first public peer to
				// contain all hidden subnets, except for the one owned by the node
				if !hasIPv4(onr) {
					continue
				}

				// Also add all other subnets if this is the pub node
				if onr.NodeId == pubNr {
					for owner, subnet := range hiddenSubnets {
						// Do not add our own subnet
						if owner == nr.NodeId {
							continue
						}

						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, *wgIP(&schema.IPRange{subnet.IPNet}))
					}

					for _, subnet := range ipv6OnlySubnets {
						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, *wgIP(&schema.IPRange{subnet.IPNet}))
					}
				}

				// Endpoint must be IPv4
				for _, pep := range onr.PubEndpoints {
					if pep.To4() != nil {
						endpoint = fmt.Sprintf("%s:%d", pep.String(), onr.WireguardListenPort)
						break
					}
				}
			} else if len(onr.PubEndpoints) == 0 && hasIPv4(*nr) {
				// if the peer is hidden but we have IPv4,  we can connect to it, but we don't know
				// an endpoint.
				endpoint = ""
			} else {
				// if we are not hidden, we add all other nodes, unless we don't
				// have IPv4, because then we also can't connect to hidden nodes.
				// Ignore hidden nodes if we don't have IPv4
				if !hasIPv4(*nr) && len(onr.PubEndpoints) == 0 {
					continue
				}

				// both nodes are public therefore we can connect over IPv6

				// if this is the selected pubNr - also need to add allowedIPs
				// for the hidden nodes
				if onr.NodeId == pubNr {
					for _, subnet := range hiddenSubnets {
						allowedIPs = append(allowedIPs, subnet)
						allowedIPs = append(allowedIPs, *wgIP(&schema.IPRange{subnet.IPNet}))
					}
				}

				// Since the node is not hidden, we know that it MUST have at least
				// 1 IPv6 address
				for _, pep := range onr.PubEndpoints {
					if pep.To4() == nil && pep.To16() != nil {
						endpoint = fmt.Sprintf("[%s]:%d", pep.String(), onr.WireguardListenPort)
						break
					}
				}

				// as a fallback assign IPv4
				if endpoint == "" {
					for _, pep := range onr.PubEndpoints {
						if pep.To4() != nil {
							endpoint = fmt.Sprintf("%s:%d", pep.String(), onr.WireguardListenPort)
							break
						}
					}
				}
			}

			// Add subnets for external access
			for i := 0; i < len(allowedIPs); i++ {
				for _, subnet := range externalSubnets[allowedIPs[i].String()] {
					allowedIPs = append(allowedIPs, schema.IPRange{subnet.IPNet})
					allowedIPs = append(allowedIPs, *wgIP(&schema.IPRange{subnet.IPNet}))
				}
			}

			nr.Peers = append(nr.Peers, workloads.WireguardPeer{
				PublicKey:      onr.WireguardPublicKey,
				Iprange:        onr.Iprange,
				AllowedIprange: allowedIPs,
				Endpoint:       endpoint,
			})
		}

		// Add configured external access peers
		for _, ea := range accessPoints[nr.NodeId] {
			allowedIPs := make([]schema.IPRange, 2)
			allowedIPs[0] = schema.IPRange{ea.Subnet.IPNet}
			allowedIPs[1] = *wgIP(&schema.IPRange{ea.Subnet.IPNet})

			nr.Peers = append(nr.Peers, workloads.WireguardPeer{
				PublicKey:      ea.WGPublicKey,
				Iprange:        schema.IPRange{ea.Subnet.IPNet},
				AllowedIprange: allowedIPs,
				Endpoint:       "",
			})
		}
	}

	return nil
}

func isIPv4Subnet(n types.IPNet) bool {
	ones, bits := n.IPNet.Mask.Size()
	if bits != 32 {
		return false
	}
	return ones <= 30
}

func genWGQuick(wgPrivateKey string, localSubnet types.IPNet, peerWgPubKey string, allowedSubnet schema.IPRange, peerEndpoint string) (string, error) {
	type data struct {
		PrivateKey    string
		Address       string
		PeerWgPubKey  string
		AllowedSubnet string
		PeerEndpoint  string
	}

	if !isIPv4Subnet(localSubnet) {
		return "", errors.New("local subnet is not a valid IPv4 subnet")
	}

	tmpl, err := template.New("wg").Parse(wgTmpl)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}

	if err := tmpl.Execute(buf, data{
		PrivateKey:    wgPrivateKey,
		Address:       wgIP(&schema.IPRange{localSubnet.IPNet}).String(),
		PeerWgPubKey:  peerWgPubKey,
		AllowedSubnet: strings.Join([]string{allowedSubnet.String(), types.NewIPNet(wgSubnet(&allowedSubnet.IPNet)).String()}, ","),
		PeerEndpoint:  peerEndpoint,
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var wgTmpl = `
[Interface]
PrivateKey = {{.PrivateKey}}
Address = {{.Address}}

[Peer]
PublicKey = {{.PeerWgPubKey}}
AllowedIPs = {{.AllowedSubnet}}
PersistentKeepalive = 20
{{if .PeerEndpoint}}Endpoint = {{.PeerEndpoint}}{{end}}
`

// NetworkGraph creates a networkgraph for a network
func (n *NetworkBuilder) NetworkGraph(w io.Writer) error {
	nodes := make(map[string]dot.Node)
	nodesByID := make(map[string]dot.Node)
	graph := dot.NewGraph(dot.Directed)

	for _, nr := range n.NetResources {
		node := graph.Node(strings.Join([]string{nr.NodeId, nr.Iprange.String()}, "\n")).Box()
		// set special style for "hidden" nodes
		if len(nr.PubEndpoints) == 0 {
			node.Attr("style", "dashed")
			node.Attr("color", "blue")
			graph.AddToSameRank("hidden nodes", node)
		}
		nodes[nr.WireguardPublicKey] = node
		nodesByID[nr.NodeId] = node
	}

	// add external access
	for _, ea := range n.AccessPoints {
		node := graph.Node(strings.Join([]string{"External network", ea.Subnet.String()}, "\n")).Box()
		// set style for hidden nodes
		node.Attr("style", "dashed")
		node.Attr("color", "green")
		graph.AddToSameRank("external access", node)
		// add link to access point
		edge := graph.Edge(node, nodesByID[ea.NodeID], n.Iprange.String())
		if ea.IP4 {
			edge.Attr("color", "blue")
		}
		nodes[ea.WGPublicKey] = node
	}

	for _, nr := range n.NetResources {
		for _, peer := range nr.Peers {
			allowedIPs := make([]string, 0, len(peer.AllowedIprange)/2)
			for _, aip := range peer.AllowedIprange {
				if !isCGN(aip) {
					allowedIPs = append(allowedIPs, aip.String())
				}
			}

			edge := graph.Edge(nodes[nr.WireguardPublicKey], nodes[peer.PublicKey], strings.Join(allowedIPs, "\n"))
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

func wgIP(subnet *schema.IPRange) *schema.IPRange {
	// example: 10.3.1.0 -> 100.64.3.1
	a := subnet.IP[len(subnet.IP)-3]
	b := subnet.IP[len(subnet.IP)-2]

	return &schema.IPRange{net.IPNet{
		IP:   net.IPv4(0x64, 0x40, a, b),
		Mask: net.CIDRMask(32, 32),
	}}
}

func wgSubnet(subnet *net.IPNet) *net.IPNet {
	// example: 10.3.1.0 -> 100.64.3.1
	a := subnet.IP[len(subnet.IP)-3]
	b := subnet.IP[len(subnet.IP)-2]

	ones, _ := subnet.Mask.Size()

	return &net.IPNet{
		IP:   net.IPv4(0x64, 0x40, a, b),
		Mask: net.CIDRMask(ones+8, 32),
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

func isCGN(subnet schema.IPRange) bool {
	_, block, err := net.ParseCIDR("100.64.0.0/10")
	if err != nil {
		panic(err)
	}
	return block.Contains(subnet.IP)
}

func (n *NetworkBuilder) extractAccessPoints() {
	// gather all actual nodes, using their wg pubkey as key in the map (NodeID
	// can't be seen in the actual peer struct)
	actualNodes := make(map[string]struct{})
	for _, nr := range n.NetResources {
		actualNodes[nr.WireguardPublicKey] = struct{}{}
	}

	aps := []AccessPoint{}
	for _, nr := range n.NetResources {
		for _, peer := range nr.Peers {
			if _, exists := actualNodes[peer.PublicKey]; !exists {
				// peer is not a node so it must be external
				aps = append(aps, AccessPoint{
					NodeID:      nr.NodeId,
					Subnet:      peer.Iprange,
					WGPublicKey: peer.PublicKey,
					// we can't infer if we use IPv6 or IPv4
				})
			}
		}
	}
	n.AccessPoints = aps
}

// a node has either a public namespace with []ipv4 or/and []ipv6 -or-
// some interface has received a SLAAC addr
// which has been registered in BCDB
func (n *NetworkBuilder) getEndPointAddrs(nodeID pkg.Identifier) ([]types.IPNet, error) {
	schemaNode, err := n.Bcdb.Directory.NodeGet(nodeID.Identity(), false)
	if err != nil {
		return nil, err
	}
	node := types.NewNodeFromSchema(schemaNode)
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
