package main

import (
	"context"
	"os/exec"

	"github.com/pkg/errors"
)

func ensureHostFw(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "nft list ruleset")
	out, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "could not load existing nft rules")
	}

	// there are already rules in place, return
	if string(out) == "" {
		return nil
	}

	cmd = exec.CommandContext(ctx, "/bin/sh", "-c",
		`nft 'add table inet filter'
nft 'add chain inet filter input   { type filter hook input priority filter; policy accept; }'
nft 'add chain inet filter forward { type filter hook input priority filter; policy accept; }'
nft 'add chain inet filter output  { type filter hook input priority filter; policy accept; }'

nft 'add table arp filter'
nft 'add chain arp filter input  { type filter hook input priority filter; policy accept; }'
nft 'add chain arp filter output { type filter hook input priority filter; policy accept; }'

nft 'add table bridge filter'
nft 'add chain bridge filter input   { type filter hook input priority filter; policy accept; }'
nft 'add chain bridge filter forward { type filter hook input priority filter; policy accept; }'
nft 'add chain bridge filter output  { type filter hook input priority filter; policy accept; }'`)

	if err = cmd.Run(); err != nil {
		return errors.Wrap(err, "could not set up host nft rules")
	}

	return nil
}
