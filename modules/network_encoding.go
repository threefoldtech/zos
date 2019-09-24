package modules

import (
	"encoding/json"
	"net"
)

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net"
// )

// var _reachabilityV4ToStr = map[ReachabilityV4]string{
// 	ReachabilityV4Public: "public",
// 	ReachabilityV4Hidden: "hidden",
// }

// var _reachabilityV4FromStr = map[string]ReachabilityV4{
// 	"public": ReachabilityV4Public,
// 	"hidden": ReachabilityV4Hidden,
// }

// var _reachabilityV6ToStr = map[ReachabilityV6]string{
// 	ReachabilityV6Public: "public",
// 	ReachabilityV6ULA:    "hidden",
// }

// var _reachabilityV6FromStr = map[string]ReachabilityV6{
// 	"public": ReachabilityV6Public,
// 	"hidden": ReachabilityV6ULA,
// }

// // UnmarshalJSON implements encoding/json.Unmarshaler
// func (n *NodeID) UnmarshalJSON(b []byte) error {
// 	tmp := struct {
// 		ID             string `json:"id"`
// 		FarmerID       string `json:"farmer_id"`
// 		ReachabilityV4 string `json:"reachability_v4"`
// 		ReachabilityV6 string `json:"reachability_v6"`
// 	}{}
// 	if err := json.Unmarshal(b, &tmp); err != nil {
// 		return err
// 	}
// 	var ok bool

// 	*n = NodeID{}
// 	n.ID = tmp.ID
// 	n.FarmerID = tmp.FarmerID
// 	if tmp.ReachabilityV4 != "" {
// 		n.ReachabilityV4, ok = _reachabilityV4FromStr[tmp.ReachabilityV4]
// 		if !ok {
// 			return fmt.Errorf("unsupported ReachabilityV4 %s", tmp.ReachabilityV4)
// 		}
// 	}
// 	if tmp.ReachabilityV6 != "" {
// 		n.ReachabilityV6, ok = _reachabilityV6FromStr[tmp.ReachabilityV6]
// 		if !ok {
// 			return fmt.Errorf("unsupported ReachabilityV4 %s", tmp.ReachabilityV4)
// 		}
// 	}

// 	return nil
// }

// // MarshalJSON implements encoding/json.Marshaler
// func (n *NodeID) MarshalJSON() ([]byte, error) {
// 	tmp := struct {
// 		ID             string `json:"id"`
// 		FarmerID       string `json:"farmer_id"`
// 		ReachabilityV4 string `json:"reachability_v4"`
// 		ReachabilityV6 string `json:"reachability_v6"`
// 	}{
// 		ID:             n.ID,
// 		FarmerID:       n.FarmerID,
// 		ReachabilityV4: _reachabilityV4ToStr[n.ReachabilityV4],
// 		ReachabilityV6: _reachabilityV6ToStr[n.ReachabilityV6],
// 	}

// 	return json.Marshal(tmp)
// }

// UnmarshalJSON implements encoding/json.Unmarshaler
func (p *Peer) UnmarshalJSON(b []byte) error {
	tmp := struct {
		Subnet      string   `json:"subnet"`
		WGPublicKey string   `json:"wg_public_key"`
		AllowedIPs  []string `json:"allowed_ips"`
		Endpoint    string   `json:"endpoint"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	var err error
	*p = Peer{}

	if tmp.Subnet != "" {
		_, p.Subnet, err = net.ParseCIDR(tmp.Subnet)
		if err != nil {
			return err
		}
	}
	p.WGPublicKey = tmp.WGPublicKey
	p.WGPublicKey = tmp.WGPublicKey
	p.Endpoint = net.ParseIP(tmp.Endpoint)

	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (p *Peer) MarshalJSON() ([]byte, error) {

	allowedIPs := make([]string, len(p.AllowedIPs))
	for i, ip := range p.AllowedIPs {
		allowedIPs[i] = ip.String()
	}
	tmp := struct {
		Subnet      string   `json:"subnet"`
		WGPublicKey string   `json:"wg_public_key"`
		AllowedIPs  []string `json:"allowed_ips"`
		Endpoint    string   `json:"endpoint"`
	}{
		Subnet:      p.Subnet.String(),
		WGPublicKey: p.WGPublicKey,
		AllowedIPs:  allowedIPs,
		Endpoint:    p.Endpoint.String(),
	}

	return json.Marshal(tmp)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (r *NetResource) UnmarshalJSON(b []byte) error {
	tmp := struct {
		NodeID       string  `json:"node_id"`
		Subnet       string  `json:"subnet"`
		WGPrivateKey string  `json:"wg_private_key"`
		WGPublicKey  string  `json:"wg_public_key"`
		WGListenPort uint16  `json:"wg_listen_port"`
		Peers        []*Peer `json:"peers"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	var err error
	*r = NetResource{}

	r.NodeID = tmp.NodeID
	r.Peers = tmp.Peers
	if tmp.Subnet != "" {
		_, r.Subnet, err = net.ParseCIDR(tmp.Subnet)
		if err != nil {
			return err
		}
	}
	r.WGPrivateKey = tmp.WGPrivateKey
	r.WGPublicKey = tmp.WGPublicKey
	r.WGListenPort = tmp.WGListenPort

	return nil
}

// MarshalJSON implements encoding/json.Unmarshaler
func (r *NetResource) MarshalJSON() ([]byte, error) {
	tmp := struct {
		NodeID       string  `json:"node_id"`
		Subnet       string  `json:"subnet"`
		Peers        []*Peer `json:"peers"`
		WGPrivateKey string  `json:"wg_private_key"`
		WGPublicKey  string  `json:"wg_public_key"`
		WGListenPort uint16  `json:"wg_listen_port"`
	}{
		NodeID:       r.NodeID,
		Subnet:       r.Subnet.String(),
		Peers:        r.Peers,
		WGPrivateKey: r.WGPrivateKey,
		WGPublicKey:  r.WGPublicKey,
		WGListenPort: r.WGListenPort,
	}
	return json.Marshal(tmp)
}

// // UnmarshalJSON implements encoding/json.Unmarshaler
// func (i *Ipv4Conf) UnmarshalJSON(b []byte) error {
// 	tmp := struct {
// 		Cidr      string `json:"cidr"`
// 		Gateway   string `json:"gateway"`
// 		Metric    uint32 `json:"metric"`
// 		Iface     string `json:"iface"`
// 		EnableNat bool   `json:"enable_nat"`
// 	}{}

// 	if err := json.Unmarshal(b, &tmp); err != nil {
// 		return err
// 	}

// 	var err error
// 	*i = Ipv4Conf{}
// 	if tmp.Cidr != "" {
// 		_, i.CIDR, err = net.ParseCIDR(tmp.Cidr)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	i.Gateway = net.ParseIP(tmp.Gateway)
// 	i.Metric = tmp.Metric
// 	i.Iface = tmp.Iface
// 	i.EnableNAT = tmp.EnableNat

// 	return nil
// }

// // MarshalJSON implements encoding/json.Unmarshaler
// func (i *Ipv4Conf) MarshalJSON() ([]byte, error) {
// 	tmp := struct {
// 		Cidr      string `json:"cidr"`
// 		Gateway   string `json:"gateway"`
// 		Metric    uint32 `json:"metric"`
// 		Iface     string `json:"iface"`
// 		EnableNat bool   `json:"enable_nat"`
// 	}{
// 		Cidr:      i.CIDR.String(),
// 		Gateway:   i.Gateway.String(),
// 		Metric:    i.Metric,
// 		Iface:     i.Iface,
// 		EnableNat: i.EnableNAT,
// 	}

// 	return json.Marshal(tmp)
// }

// // UnmarshalJSON implements encoding/json.Unmarshaler
// func (i *Ipv6Conf) UnmarshalJSON(b []byte) error {
// 	tmp := struct {
// 		Addr    string `json:"addr"`
// 		Gateway string `json:"gateway"`
// 		Metric  uint32 `json:"metric"`
// 		Iface   string `json:"iface"`
// 	}{}

// 	if err := json.Unmarshal(b, &tmp); err != nil {
// 		return err
// 	}

// 	*i = Ipv6Conf{}
// 	if tmp.Addr != "" {
// 		ip, ipnet, err := net.ParseCIDR(tmp.Addr)
// 		if err != nil {
// 			return err
// 		}
// 		ipnet.IP = ip
// 		i.Addr = ipnet
// 	}
// 	i.Gateway = net.ParseIP(tmp.Gateway)
// 	i.Metric = tmp.Metric
// 	i.Iface = tmp.Iface
// 	return nil
// }

// // MarshalJSON implements encoding/json.Unmarshaler
// func (i *Ipv6Conf) MarshalJSON() ([]byte, error) {
// 	tmp := struct {
// 		Addr    string `json:"addr"`
// 		Gateway string `json:"gateway"`
// 		Metric  uint32 `json:"metric"`
// 		Iface   string `json:"iface"`
// 	}{
// 		Addr:    i.Addr.String(),
// 		Gateway: i.Gateway.String(),
// 		Metric:  i.Metric,
// 		Iface:   i.Iface,
// 	}

// 	return json.Marshal(tmp)
// }

// // UnmarshalJSON implements encoding/json.Unmarshaler
// func (d *DNAT) UnmarshalJSON(b []byte) error {
// 	tmp := struct {
// 		InternalIP   string `json:"internal_ip"`
// 		InternalPort uint16 `json:"internal_port"`
// 		ExternalIP   string `json:"external_ip"`
// 		ExternalPort uint16 `json:"external_port"`
// 		Protocol     string `json:"protocol"`
// 	}{}

// 	if err := json.Unmarshal(b, &tmp); err != nil {
// 		return err
// 	}

// 	*d = DNAT{}
// 	d.InternalIP = net.ParseIP(tmp.InternalIP)
// 	d.InternalPort = tmp.InternalPort
// 	d.ExternalIP = net.ParseIP(tmp.ExternalIP)
// 	d.ExternalPort = tmp.ExternalPort
// 	d.Protocol = tmp.Protocol

// 	return nil
// }

// // MarshalJSON implements encoding/json.Unmarshaler
// func (d *DNAT) MarshalJSON() ([]byte, error) {
// 	tmp := struct {
// 		InternalIP   string `json:"internal_ip"`
// 		InternalPort uint16 `json:"internal_port"`
// 		ExternalIP   string `json:"external_ip"`
// 		ExternalPort uint16 `json:"external_port"`
// 		Protocol     string `json:"protocol"`
// 	}{
// 		InternalIP:   d.InternalIP.String(),
// 		InternalPort: d.InternalPort,
// 		ExternalIP:   d.InternalIP.String(),
// 		ExternalPort: d.ExternalPort,
// 		Protocol:     d.Protocol,
// 	}

// 	return json.Marshal(tmp)
// }

// // UnmarshalJSON implements encoding/json.Unmarshaler
// func (e *ExitPoint) UnmarshalJSON(b []byte) error {
// 	tmp := struct {
// 		Ipv4Conf  *Ipv4Conf    `json:"ipv4_conf"`
// 		Ipv4Dnat  []*DNAT      `json:"ipv4_dnat"`
// 		Ipv6Conf  *Ipv6Conf    `json:"ipv6_conf"`
// 		Ipv6Allow []*Ipv6Allow `json:"ipv6_allow"`
// 	}{}

// 	if err := json.Unmarshal(b, &tmp); err != nil {
// 		return err
// 	}

// 	*e = ExitPoint{}
// 	e.Ipv4Conf = tmp.Ipv4Conf
// 	e.Ipv4DNAT = tmp.Ipv4Dnat
// 	e.Ipv6Conf = tmp.Ipv6Conf
// 	e.Ipv6Allow = tmp.Ipv6Allow

// 	return nil
// }

// // MarshalJSON implements encoding/json.Marshaler
// func (e *ExitPoint) MarshalJSON() ([]byte, error) {
// 	tmp := struct {
// 		Ipv4Conf  *Ipv4Conf    `json:"ipv4_conf"`
// 		Ipv4Dnat  []*DNAT      `json:"ipv4_dnat"`
// 		Ipv6Conf  *Ipv6Conf    `json:"ipv6_conf"`
// 		Ipv6Allow []*Ipv6Allow `json:"ipv6_allow"`
// 	}{
// 		Ipv4Conf:  e.Ipv4Conf,
// 		Ipv4Dnat:  e.Ipv4DNAT,
// 		Ipv6Conf:  e.Ipv6Conf,
// 		Ipv6Allow: e.Ipv6Allow,
// 	}
// 	var ips []*Ipv6Allow
// 	for i, ip := range e.Ipv6Allow {
// 		d := ip.Ipv6Dest
// 		p := ip.Port
// 		ips[i] = &Ipv6Allow{Ipv6Dest: d, Port: p}
// 	}
// 	tmp.Ipv6Allow = ips
// 	return json.Marshal(tmp)
// }

// UnmarshalJSON implements encoding/json.Unmarshaler
func (d *Network) UnmarshalJSON(b []byte) (err error) {
	tmp := struct {
		Name         string         `json:"name,omitempty"`
		NetID        string         `json:"net_id,omitempty"`
		NetResources []*NetResource `json:"net_resources,omitempty"`
		IPRange      string         `json:"ip_range,omitempty"`
	}{}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	*d = Network{}
	d.NetID = NetID(tmp.NetID)
	d.NetResources = tmp.NetResources
	if tmp.IPRange != "" {
		_, d.IPRange, err = net.ParseCIDR(tmp.IPRange)
		if err != nil {
			return err
		}
	}

	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (d *Network) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Name         string         `json:"name,omitempty"`
		NetID        string         `json:"net_id,omitempty"`
		NetResources []*NetResource `json:"net_resources,omitempty"`
		IPRange      string         `json:"ip_range,omitempty"`
	}{
		NetID:        string(d.NetID),
		NetResources: d.NetResources,
	}
	if d.IPRange != nil {
		tmp.IPRange = d.IPRange.String()
	}
	return json.Marshal(tmp)
}
