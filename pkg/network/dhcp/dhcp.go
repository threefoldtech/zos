package dhcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os/exec"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type ProbeOutput struct {
	Subnet        string `json:"subnet"`
	Router        string `json:"router"`
	IP            string `json:"ip"`
	SourceAddress string `json:"siaddr"`
	DNS           string `json:"dns"`
	ServerID      string `json:"serverid"`
	BroadcastIP   string `json:"broadcast"`
	Lease         string `json:"lease"`
}

func (p *ProbeOutput) IPNet() (*net.IPNet, error) {
	mask := net.ParseIP(p.Subnet).To4()
	if mask == nil {
		return nil, fmt.Errorf("invalid subnet mask (%s)", p.Subnet)
	}
	ip := net.ParseIP(p.IP).To4()
	if ip == nil {
		return nil, fmt.Errorf("invalid ip  (%s)", p.IP)
	}

	return &net.IPNet{
		IP:   ip,
		Mask: net.IPMask(mask),
	}, nil
}

func Probe(ctx context.Context, inf string) (output ProbeOutput, err error) {
	// use udhcpc to probe the interface.
	// this depends on that the interface is UP

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	t := 10
	check := func() error {
		args := []string{
			"-i", inf, // the interface to prope
			"-q",      // exit once lease is optained
			"-f",      // foreground
			"-T", "1", // pauses for a second between packets timeout
			"-t", fmt.Sprint(t), // send 't' dhcp discover packets
			"-s", "/usr/share/udhcp/probe.script", // use the probe script
			"--now", // exit if lease is not obtained
		}

		cmd := exec.CommandContext(ctx, "udhcpc", args...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		t += 5
		return cmd.Run()
	}

	if err := backoff.Retry(check, backoff.NewExponentialBackOff()); err != nil {
		return output, errors.Wrapf(err, "failed to probe interface '%s': %s", inf, stderr.String())
	}

	log.Debug().Str("output", stdout.String()).Msg("output from dhcp proping")
	dec := json.NewDecoder(&stdout)
	if err := dec.Decode(&output); err != nil {
		buf, _ := io.ReadAll(dec.Buffered())
		str := stdout.String() + string(buf)
		return output, errors.Wrapf(err, "failed to decode proper output (%s)", str)
	}

	log.Debug().Msgf("output: %+v", output)
	return
}
