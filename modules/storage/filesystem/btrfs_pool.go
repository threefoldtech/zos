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

// BtrfsList lists all availabel btrfs pools
func BtrfsList(ctx context.Context) ([]Btrfs, error) {
	output, err := run(ctx, "btrfs", "filesystem", "show", "--raw")
	if err != nil {
		return nil, err
	}

	return parseList(string(output))
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
