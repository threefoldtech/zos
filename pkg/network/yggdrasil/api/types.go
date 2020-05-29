package yggdrasil

type NodeInfo struct {
	BoxPubKey    string `json:"box_pub_key"`
	BuildName    string `json:"build_name"`
	BuildVersion string `json:"build_version"`
	Coords       string `json:"coords"`
	IPv6Addr     string `json:"ipv6_address"`
	Subnet       string `json:"subnet"`
}

type Peer struct {
	IPv6Addr   string  `json:"ipv6_address"`
	BytesRecvd int     `json:"bytes_recvd"`
	BytesSent  int     `json:"bytes_sent"`
	Endpoint   string  `json:"endpoint"`
	Port       int     `json:"port"`
	Proto      string  `json:"proto"`
	Uptime     float64 `json:"uptime"`
}
