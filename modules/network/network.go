package network

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"time"

	"github.com/vishvananda/netlink"
)

const carrierFile = "/sys/class/net/%s/carrier"

func interfaces() ([]netlink.Link, error) {
	return netlink.LinkList()
}

func filterBridge(links []netlink.Link) []*netlink.Bridge {
	bridges := []*netlink.Bridge{}

	for _, link := range links {
		if link.Type() == "bridge" {
			bridge, ok := link.(*netlink.Bridge)
			if ok {
				bridges = append(bridges, bridge)
			}
		}
	}
	return bridges
}

func isPlugged(inf string) bool {
	data, err := ioutil.ReadFile(fmt.Sprintf(carrierFile, inf))
	if err != nil {
		return false
	}
	data = bytes.TrimSpace(data)
	if string(data) != "1" {
		return false
	}

	return true
}

//dhcpProbe will do a dhcp request on the interface inf
// if the interface gets a lease from the dhcp server, dhcpProbe return true and a nil error
// if something unexpected happens a non nil error is return
// if the interface didn't receive an lease, false and a nil error is returns
func dhcpProbe(inf string) (bool, error) {
	cmd := exec.Command("udhcpc",
		"-f", //foreground
		"-i", inf,
		"-t", "3", //try 10 times before giving up
		"-A", "3", //wait 3 seconds between each trial
		"-s", "/usr/share/udhcp/simple.script",
		"--now", // exit if lease is not optained
	)

	cmd.Start()
	// time.Sleep(time.Second * 9)

	link, err := netlink.LinkByName(inf)
	if err != nil {
		// TODO
		return false, err
	}

	var hasGW = false
	for hasGW == false {
		hasGW, err = hasDefaultGW(link)
		if err != nil {
			return false, err
		}

		time.Sleep(1 * time.Second)
	}

	if !cmd.ProcessState.Exited {
		cmd.Process.Kill(os.SIGKILL)
	}

	return hasGW, nil
}

func hasDefaultGW(link netlink.Link) (bool, error) {

	addrs, err := netlink.AddrList(link, netlink.FAMILY_V4|netlink.FAMILY_V6)
	if err != nil {
		return false, err
	}

	if len(addrs) <= 0 {
		return false, nil
	}

	routes, err := netlink.RouteList(link, netlink.FAMILY_V4|netlink.FAMILY_V6)
	if err != nil {
		return false, err
	}

	if !routes[0].Dst.IP.Equal(net.IP("0.0.0.0/0")) && !routes[0].Dst.IP.Equal(net.IP("::")) {
		return false, nil
	}

	return true, nil
}

// func filterLinkUp(links []netlink.Link) []netlink.Link {
// 	links := []*netlink.Link{}

// 	for _, link := range links {

// 		if link.Type() == "bridge" {
// 			links, ok := link.(*netlink.Bridge)
// 			if ok {
// 				links = append(links, link)
// 			}
// 		}
// 	}
// 	return links
// }
