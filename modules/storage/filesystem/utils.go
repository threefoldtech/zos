package filesystem

import (
	"bufio"
	"fmt"
	"os"
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

// BindMount remounts an existing directory in a given target using the mount
// syscall with the BIND flag set
func BindMount(src Volume, target string) error {
	return syscall.Mount(src.Path(), target, "btrfs", syscall.MS_BIND, "")
}
