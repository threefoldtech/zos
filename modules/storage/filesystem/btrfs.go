package filesystem

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/threefoldtech/zosv2/modules"
)

var (
	_ Filesystem = (*btrfs)(nil)
)

// btrfs is the filesystem implementation for btrfs
type btrfs struct{}

func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s", string(err.Stderr))
		}
	}

	return output, nil
}

func (b *btrfs) btrfs(ctx context.Context, args ...string) ([]byte, error) {
	return run(ctx, "btrfs", args...)
}

func (b *btrfs) Create(ctx context.Context, name string, devices []string, policy modules.RaidProfile) (Pool, error) {
	block, err := Devices(ctx)
	if err != nil {
		return nil, err
	}

	for _, device := range block {
		if device.Label == name {
			return nil, fmt.Errorf("unique name is required")
		}
	}

	args := []string{
		"-L", name,
		"-d", string(policy),
		"-m", string(policy),
	}

	args = append(args, devices...)
	if _, err := run(ctx, "mkfs.btrfs", args...); err != nil {
		return nil, err
	}

	return btrfsPool(name), nil
}

func (b *btrfs) List(ctx context.Context) ([]Btrfs, error) {
	output, err := b.btrfs(ctx, "filesystem", "show", "--raw")
	if err != nil {
		return nil, err
	}

	return parseList(string(output))
}

type btrfsPool string

func (p btrfsPool) Mounted() (string, bool) {
	return "", false
}

func (p btrfsPool) Mount() (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (p btrfsPool) UnMount() error {
	return fmt.Errorf("not implemented")
}

func (p btrfsPool) AddDevice(device string) error {
	return fmt.Errorf("not implemented")
}

func (p btrfsPool) RemoveDevice(device string) error {
	return fmt.Errorf("not implemented")
}

func (p btrfsPool) Volumes() ([]Volume, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p btrfsPool) AddVolume(size uint64) (Volume, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p btrfsPool) RemoveVolume(volume Volume) error {
	return fmt.Errorf("not implemented")
}
