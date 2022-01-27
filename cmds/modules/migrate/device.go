package migrate

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/pkg/errors"
)

/*
{
   "blockdevices": [
      {"path":"/dev/sda", "type":"disk", "subsystems":"block:scsi:pci", "mountpoint":"/mnt/fe33fecf-cc24-4b81-89fd-f9c991200c4b"},
      {"path":"/dev/sdb", "type":"disk", "subsystems":"block:scsi:usb:pci", "mountpoint":null},
      {"path":"/dev/sdb1", "type":"part", "subsystems":"block:scsi:usb:pci", "mountpoint":null}
   ]
}

*/
type Device struct {
	Path       string `json:"name"`
	Type       string `json:"type"`
	Subsystems string `json:"subsystems"`
	Mountpoint string `json:"mountpoint"`
}

type Filter func(d *Device) bool

func IsUsb(d *Device) bool {
	return d.Subsystems == "block:scsi:usb:pci"
}

func Not(f Filter) Filter {
	return func(d *Device) bool {
		return !f(d)
	}
}

func devices(ctx context.Context, filter ...Filter) ([]Device, error) {
	cmd := exec.CommandContext(ctx, "lsblk", "-o", "name,type,subsystems,mountpoint", "--json", "--exclude", "1,2,11", "--path")
	bytes, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list devices on node")
	}

	var output struct {
		Devices []Device `json:"blockdevices"`
	}

	if err := json.Unmarshal(bytes, &output); err != nil {
		return nil, errors.Wrap(err, "failed to load devices informations")
	}

	if len(filter) == 0 {
		return output.Devices, nil
	}

	filtered := output.Devices[:0]
next:
	for _, device := range output.Devices {
		for _, f := range filter {
			if !f(&device) {
				continue next
			}
		}

		filtered = append(filtered, device)
	}

	return filtered, nil
}
