package network

// import (
// 	"bytes"
// 	"fmt"
// 	"io/ioutil"
// 	"os"

// 	"github.com/threefoldtech/0-core/base/mgr"
// 	"github.com/threefoldtech/0-core/base/pm"
// )

// const (
// 	ProtocolDHCP = "dhcp"

// 	carrierFile = "/sys/class/net/%s/carrier"
// )

// func init() {
// 	protocols[ProtocolDHCP] = &dhcpProtocol{}
// }

// type dhcpProtocol struct {
// }

// func (d *dhcpProtocol) isPlugged(inf string) error {
// 	data, err := ioutil.ReadFile(fmt.Sprintf(carrierFile, inf))
// 	if err != nil {
// 		return err
// 	}
// 	data = bytes.TrimSpace(data)
// 	if string(data) == "1" {
// 		return nil
// 	}

// 	return fmt.Errorf("interface %s has no carrier(%s)", inf, string(data))
// }

// func (d *dhcpProtocol) Configure(_ NetworkManager, inf string) error {
// 	// if err := d.isPlugged(inf); err != nil {
// 	// 	return err
// 	// }

// 	hostname, _ := os.Hostname()
// 	hostid := fmt.Sprintf("hostname:%s", hostname)

// 	cmd := &pm.Command{
// 		ID:      fmt.Sprintf("udhcpc/%s", inf),
// 		Command: pm.CommandSystem,
// 		Arguments: pm.MustArguments(
// 			pm.SystemCommandArguments{
// 				Name: "udhcpc",
// 				Args: []string{
// 					"-f", //foreground
// 					"-i", inf,
// 					"-t", "10", //try 10 times before giving up
// 					"-A", "3", //wait 3 seconds between each trial
// 					"-s", "/usr/share/udhcp/simple.script",
// 					"--now",      // exit if lease is not optained
// 					"-x", hostid, //set hostname on dhcp request
// 				},
// 			},
// 		),
// 	}

// 	if _, err := mgr.Run(cmd); err != nil {
// 		return err
// 	}

// 	return nil
// }
