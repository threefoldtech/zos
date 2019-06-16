package filesystem

import (
	"context"
	"fmt"
	"strings"
)

// Btrfs holds metadata of underlying btrfs filesystem
type Btrfs struct {
	Label        string        `json:"label"`
	UUID         string        `json:"uuid"`
	TotalDevices int           `json:"total_devices"`
	Used         int64         `json:"used"`
	Devices      []BtrfsDevice `json:"devices"`
	Warnings     string        `json:"warnings"`
}

// BtrfsDevice holds metadata about a single device in a btrfs filesystem
type BtrfsDevice struct {
	Missing bool   `json:"missing,omitempty"`
	DevID   int    `json:"dev_id"`
	Size    int64  `json:"size"`
	Used    int64  `json:"used"`
	Path    string `json:"path"`
}

// BtrfsVolume holds metadata about a single subvolume
type BtrfsVolume struct {
	Path       string
	ID         uint64
	Generation uint64
	ParentID   uint64
}

// BtrfsList lists all availabel btrfs pools
// if label is provided, only get fs of that label, if mounted = True, only return
// mounted filesystems, otherwise any.
func BtrfsList(ctx context.Context, label string, mounted bool) ([]Btrfs, error) {
	args := []string{
		"filesystem", "show", "--raw",
	}

	if mounted {
		args = append(args, "-m")
	}

	if len(label) != 0 {
		args = append(args, label)
	}

	output, err := run(ctx, "btrfs", args...)
	if err != nil {
		return nil, err
	}

	return parseList(string(output))
}

// BtrfsSubvolumeList list direct subvolumes of this location
func BtrfsSubvolumeList(ctx context.Context, root string) ([]BtrfsVolume, error) {
	output, err := run(ctx, "btrfs", "subvolume", "list", "-o", root)
	if err != nil {
		return nil, err
	}

	var volumes []BtrfsVolume
	for _, line := range strings.Split(string(output), "\n") {
		// ID 263 gen 45 top level 261 path root/home
		var volume BtrfsVolume
		if _, err := fmt.Sscanf(line, "ID %d gen %d top level %d path %s",
			&volume.ID, &volume.Generation, &volume.ParentID, &volume.Path); err != nil {
			return nil, err
		}

		volumes = append(volumes, volume)
	}

	return volumes, nil
}

func parseList(output string) ([]Btrfs, error) {
	var fss []Btrfs

	blocks := strings.Split(output, "\n\n")
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		// Ensure that fsLines starts with Label (and collect all warnings into fs.Warnings)
		labelIdx := strings.Index(block, "Label:")
		if labelIdx != 0 {
			block = block[labelIdx:]
		}
		fsLines := strings.Split(block, "\n")
		if len(fsLines) < 3 {
			continue
		}
		fs, err := parseFS(fsLines)

		if err != nil {
			return fss, err
		}
		fss = append(fss, fs)
	}
	return fss, nil
}

func parseFS(lines []string) (Btrfs, error) {
	// first line should be label && uuid
	var label, uuid string
	_, err := fmt.Sscanf(lines[0], `Label: %s uuid: %s`, &label, &uuid)
	if err != nil {
		return Btrfs{}, err
	}
	if label != "none" {
		label = label[1 : len(label)-1]
	}

	// total device & byte used
	var totDevice int
	var used int64
	if _, err := fmt.Sscanf(strings.TrimSpace(lines[1]), "Total devices %d FS bytes used %d", &totDevice, &used); err != nil {
		return Btrfs{}, err
	}
	var validDevsLines []string
	var fsWarnings string
	for _, line := range lines[2:] {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "**") {
			// a warning
			fsWarnings += trimmedLine
		} else {
			validDevsLines = append(validDevsLines, line)
		}
	}
	devs, err := parseDevices(validDevsLines)
	if err != nil {
		return Btrfs{}, err
	}
	return Btrfs{
		Label:        label,
		UUID:         uuid,
		TotalDevices: totDevice,
		Used:         used,
		Devices:      devs,
		Warnings:     fsWarnings,
	}, nil
}

func parseDevices(lines []string) ([]BtrfsDevice, error) {
	var devs []BtrfsDevice
	for _, line := range lines {
		if line == "" {
			continue
		}
		var dev BtrfsDevice
		if _, err := fmt.Sscanf(strings.TrimSpace(line), "devid    %d size %d used %d path %s", &dev.DevID, &dev.Size, &dev.Used, &dev.Path); err == nil {
			devs = append(devs, dev)
		}
	}
	return devs, nil
}
