package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/app"
)

type options struct {
	from      uint64
	mnemonics string
	dry       bool
	url       string
}

func run(opt options) error {
	sudo, err := substrate.NewIdentityFromSr25519Phrase(opt.mnemonics)
	if err != nil {
		return errors.Wrap(err, "failed to create identity from mnemonics")
	}

	cl, err := substrate.NewSubstrate(opt.url)
	if err != nil {
		return err
	}
	to, err := cl.GetLastNodeID()
	if err != nil {
		return errors.Wrap(err, "failed to find last node id")
	}
	log.Debug().Uint32("node-id", to).Msg("scan to last node id")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := cl.ScanNodes(ctx, uint32(opt.from), to)
	if err != nil {
		return errors.Wrap(err, "failed to start node scanning")
	}

	for scanned := range ch {
		if scanned.Err != nil {
			log.Error().Err(scanned.Err).Msgf("error while getting node: %d", scanned.ID)
			continue
		}
		node := scanned.Node
		log := log.With().
			Uint32("node-id", uint32(node.ID)).
			Uint32("farm-id", uint32(node.FarmID)).
			Bool("secure-boot", node.SecureBoot).
			Bool("certified", node.CertificationType.IsCertified).
			Logger()

		log.Info().Msg("node found")

		if !node.SecureBoot || node.CertificationType.IsCertified {
			// notthing to do anyway
			continue
		}

		log.Info().Msg("possible node to certify")
		if opt.dry {
			continue
		}

		if err := cl.SetNodeCertificate(sudo, uint32(node.ID), substrate.CertificationType{IsCertified: true}); err != nil {
			return errors.Wrap(err, "failed to mark node as certified")
		}
	}

	return nil
}

func main() {
	app.Initialize()
	var opt options

	flag.StringVar(&opt.url, "substrate", "wss://tfchain.dev.grid.tf", "chain url")
	flag.BoolVar(&opt.dry, "dry-run", false, "print the list of the nodes to be migrated")
	flag.Uint64Var(&opt.from, "from", 1, "start scanning nodes with id")
	flag.StringVar(&opt.mnemonics, "mnemonics", "", "mnemonics for the sudo key")
	flag.Parse()

	if len(opt.mnemonics) == 0 {
		fmt.Fprintln(os.Stderr, "mnemonics is required")
		os.Exit(1)
	}

	if err := run(opt); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

}
