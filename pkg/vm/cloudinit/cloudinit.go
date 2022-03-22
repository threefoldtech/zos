package cloudinit

type marsh map[string]interface{}

type NetworkObjectType string
type SubnetType string

const (
	NetworkObjectTypePhysical = "physical"
	NetworkObjectTypeRoute    = "route"

	SubnetTypeDHCP   = "dhcp"
	SubnetTypeStatic = "static"
)

type NetworkObject interface {
	ObjectType() NetworkObjectType
}

type Subnet interface {
	SubnetType() SubnetType
}

type PhysicalInterface struct {
	Name       string
	MacAddress string
	Subnets    []Subnet
}

func (p PhysicalInterface) ObjectType() NetworkObjectType {
	return NetworkObjectTypePhysical
}

func (p PhysicalInterface) MarshalYAML() (interface{}, error) {
	return marsh{
		"type":    p.ObjectType(),
		"name":    p.Name,
		"mac":     p.MacAddress,
		"subnets": p.Subnets,
	}, nil
}

type SubnetDHCP struct {
}

func (s SubnetDHCP) SubnetType() SubnetType {
	return SubnetTypeDHCP
}

func (s SubnetDHCP) MarshalYAML() (interface{}, error) {
	return marsh{
		"type": s.SubnetType(),
	}, nil
}

type SubnetStatic struct {
	Address     string
	Gateway     string
	Nameservers []string
}

func (s SubnetStatic) SubnetType() SubnetType {
	return SubnetTypeStatic
}

func (s SubnetStatic) MarshalYAML() (interface{}, error) {
	return marsh{
		"type":            s.SubnetType(),
		"address":         s.Address,
		"gateway":         s.Gateway,
		"dns_nameservers": s.Nameservers,
	}, nil
}

type Route struct {
	Destination string
	Gateway     string
	Metric      int
}

func (r Route) ObjectType() NetworkObjectType {
	return NetworkObjectTypeRoute
}

func (r Route) MarshalYAML() (interface{}, error) {
	return marsh{
		"type":        r.ObjectType(),
		"destination": r.Destination,
		"gateway":     r.Gateway,
		"metric":      r.Metric,
	}, nil
}

type Metadata struct {
	InstanceID string `yaml:"instance-id"`
	Hostname   string `yaml:"local-hostname"`
}

type User struct {
	Name string   `yaml:"name"`
	Keys []string `yaml:"ssh_authorized_keys"`
}

type Mount struct {
	Source string
	Target string
}

func (m Mount) MarshalYAML() (interface{}, error) {
	return []string{
		m.Source,
		m.Target,
		"auto",
		"default",
		"0",
		"0",
	}, nil
}

type Extension struct {
	Entrypoint  string
	Environment map[string]string
}

type Configuration struct {
	Metadata  Metadata
	Network   []NetworkObject
	Users     []User
	Mounts    []Mount
	Extension Extension
}
