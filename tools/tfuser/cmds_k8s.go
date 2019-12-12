package main

import (
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/urfave/cli"
)

func generateK8S(c *cli.Context) error {

	var (
		token  = c.String("token")
		master = c.String("master")
	)

	container := provision.K8sCluster{
		MasterAddr: master,
		Token:      token,
	}

	p, err := embed(container, provision.KubernetesReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), p)
}
