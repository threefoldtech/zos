package cloudinit

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func ExampleConfiguration() {
	config := Configuration{
		Network: []NetworkObject{
			PhysicalInterface{
				Name:       "eth0",
				MacAddress: "11:22:33:44:55:66",
				Subnets: []Subnet{
					SubnetDHCP{},
					SubnetStatic{
						Address: "192.168.1.100/24",
						Gateway: "192.168.1.1",
						Nameservers: []string{
							"1.1.1.1",
						},
					},
				},
			},
			Route{
				Destination: "default",
				Gateway:     "192.168.1.1",
				Metric:      3,
			},
		},
	}

	bytes, err := yaml.Marshal(config.Network)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(bytes))
	// Output:
	// - mac: 11:22:33:44:55:66
	//   name: eth0
	//   subnets:
	//     - type: dhcp
	//     - address: 192.168.1.100/24
	//       dns_nameservers:
	//         - 1.1.1.1
	//       gateway: 192.168.1.1
	//       type: static
	//   type: physical
	// - destination: default
	//   gateway: 192.168.1.1
	//   metric: 3
	//   type: route
}
