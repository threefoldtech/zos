package filesystem

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// getMountTarget returns the mount target of a device or false if the
// device is not mounted.
// panic, it panics if it can't read /proc/mounts
func getMountTarget(device string) (string, bool) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		panic(fmt.Errorf("failed to read /proc/mounts: %s", err))
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if fields[0] == device {
			return fields[1], true
		}
	}

	return "", false
}

// IsPathMounted checks if a path is mounted on a disk
func IsPathMounted(path string) bool {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		panic(fmt.Errorf("failed to read /proc/mounts: %s", err))
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if fields[1] == path {
			return true
		}
	}

	return false
}

// BindMount remounts an existing directory in a given target using the mount
// syscall with the BIND flag set
func BindMount(src Volume, target string) error {
	return syscall.Mount(src.Path(), target, src.FsType(), syscall.MS_BIND, "")
}

type executer interface {
	run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type executerFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

func (e executerFunc) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return e(ctx, name, args...)
}

func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s", string(err.Stderr))
		}
		return nil, err
	}

	return output, nil
}
