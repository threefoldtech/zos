package capacity

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Device struct {
	ID   uint16
	Name string
}

type Vendor struct {
	ID      uint16
	Name    string
	Devices map[uint16]Device
}

var (
	//go:embed pci/pci.ids
	src []byte

	pattern = regexp.MustCompile(`^(\t*)([\d|a-f]{4})(.+)$`)
	data    map[uint16]Vendor
)

func init() {
	var err error
	data, err = loadVendors(src)
	if err != nil {
		panic(fmt.Sprintf("failed to build vendors db: %s", err))
	}
}

func loadVendors(src []byte) (map[uint16]Vendor, error) {
	vendors := make(map[uint16]Vendor)

	scanner := bufio.NewScanner(bytes.NewBuffer(src))
	var current *Vendor
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		matches := pattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		// this should have `4 parts`
		tabs := matches[1]
		id64, err := strconv.ParseUint(matches[2], 16, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vendor or device id: %w", err)
		}
		rest := strings.TrimSpace(matches[3])

		id := uint16(id64)
		// the line has no leading tabs. this is a new vendor
		if len(tabs) == 0 {
			if current != nil {
				vendors[current.ID] = *current
			}
			current = &Vendor{
				ID:      id,
				Name:    rest,
				Devices: make(map[uint16]Device),
			}
		} else if len(tabs) == 1 {
			// that is a device
			if current == nil {
				return nil, fmt.Errorf("device appeared before a vendor: %s", line)
			}

			current.Devices[id] = Device{
				ID:   id,
				Name: rest,
			}
		}
	}

	return vendors, nil
}

// GetVendor gets a vendor object given Vendor ID
func GetVendor(vendor uint16) (Vendor, bool) {
	v, ok := data[vendor]
	return v, ok
}

// GetDevice looks up the devices db to given Vendor and Device IDs
func GetDevice(vendor uint16, device uint16) (v Vendor, d Device, ok bool) {
	v, ok = data[vendor]
	if !ok {
		return
	}
	d, ok = v.Devices[device]

	return
}

type Filter func(pci *PCI) bool

type PCI struct {
	Slot   string
	Vendor uint16
	Device uint16
	Class  uint32
}

// GetDevice gets the attached PCI device information (vendor and device)
func (p *PCI) GetDevice() (Vendor, Device, bool) {
	return GetDevice(p.Vendor, p.Device)
}

// ShortID returns a short identification string
// for the device in the format `vendor:device`
func (p *PCI) ShortID() string {
	return fmt.Sprintf("%04x:%04x", p.Vendor, p.Class)
}

func (p PCI) String() string {
	return fmt.Sprintf("%s %04x: [%04x:%04x]", p.Slot, p.Class, p.Vendor, p.Device)
}

func ListPCI(filter ...Filter) ([]PCI, error) {
	const (
		dir        = "/sys/bus/pci/devices"
		classFile  = "class"
		vendorFile = "vendor"
		deviceFile = "device"
	)

	var devices []PCI
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sys pci devices: %w", err)
	}

next:
	for _, entry := range entries {
		slot := entry.Name()
		class, err := readUint64(filepath.Join(dir, slot, classFile), 32)
		if err != nil {
			return nil, fmt.Errorf("failed to get device '%s' class: %w", slot, err)
		}
		vendor, err := readUint64(filepath.Join(dir, slot, vendorFile), 16)
		if err != nil {
			return nil, fmt.Errorf("failed to get device '%s' class: %w", slot, err)
		}

		device, err := readUint64(filepath.Join(dir, slot, deviceFile), 16)
		if err != nil {
			return nil, fmt.Errorf("failed to get device '%s' class: %w", slot, err)
		}

		pci := PCI{
			Slot:   slot,
			Class:  uint32(class),
			Vendor: uint16(vendor),
			Device: uint16(device),
		}

		for _, f := range filter {
			if !f(&pci) {
				continue next
			}
		}

		devices = append(devices, pci)
	}

	return devices, nil
}

func readUint64(path string, bitSize int) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read path '%s': %w", path, err)
	}

	return strconv.ParseUint(
		strings.TrimPrefix(
			strings.TrimSpace(string(data)),
			"0x"),
		16,
		bitSize)
}

// GPU Filter only devices with GPU capabilities
func GPU(p *PCI) bool {
	return p.Class == 0x030000
}
