package main

import (
	"encoding/hex"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/urfave/cli"
)

func generateKubernetes(c *cli.Context) error {
	var (
		cpu             = c.Uint64("cpu")
		memory          = c.Uint64("memory")
		diskSize        = c.Uint64("disk-size")
		netID           = c.String("networkd-id")
		ipString        = c.String("ip")
		plainSecret     = c.String("secret")
		nodeID          = c.String("node")
		masterIPStrings = c.StringSlice("master-ips")
		sshKeys         = c.StringSlice("ssh-keys")
	)

	if cpu == 0 || cpu > 32 {
		return errors.New("Invalid amount of vCpu for vm")
	}

	if memory == 0 {
		return errors.New("Invalid amount of memory for vm")
	}

	if diskSize == 0 {
		return errors.New("Invalid disk size for vm")
	}

	if netID == "" {
		return errors.New("VM requires a network to run in")
	}

	ip := net.ParseIP(ipString)
	if ip.To4() == nil {
		return errors.New("bad IP for vm")
	}

	if plainSecret == "" {
		return errors.New("A secret is required for kubernetes")
	}

	pk, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		return errors.Wrap(err, "failed to parse nodeID")
	}

	encrypted, err := crypto.Encrypt([]byte(plainSecret), pk)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt private key")
	}
	encryptedSecret := hex.EncodeToString(encrypted)

	masterIPs := make([]net.IP, len(masterIPStrings))
	for i, mips := range masterIPStrings {
		mip := net.ParseIP(mips)
		if mip.To4() == nil {
			return errors.New("bad master IP for vm")
		}
		masterIPs[i] = mip
	}

	kube := provision.Kubernetes{
		CPUCount:      uint8(cpu),
		Memory:        memory,
		DiskSize:      diskSize,
		NetworkID:     pkg.NetID(netID),
		IP:            ip,
		ClusterSecret: encryptedSecret,
		MasterIPs:     masterIPs,
		SSHKeys:       sshKeys,
	}

	p, err := embed(kube, provision.KubernetesReservation)
	if err != nil {
		return errors.Wrap(err, "could not generate reservation schema")
	}

	return output(c.GlobalString("output"), p)
}
