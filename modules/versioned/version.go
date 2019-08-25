package versioned

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	pattern = regexp.MustCompile(`^v(\d+).(\d+).(\d+)(\w*)$`)
)

// Version struct
type Version struct {
	major uint16
	minor uint16
	build uint16
	tag   string
}

// String formats version as 'v<Major>.<Minor>.<Build><Tag>'
func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d%s", v.major, v.minor, v.build, v.tag)
}

// MarshalText convert version object to text
func (v Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *Version) value() uint64 {
	return uint64(v.major)<<32 | uint64(v.minor)<<16 | uint64(v.build)
}

// Compare version
func (v Version) Compare(o Version) int {
	self := v.value()
	other := o.value()
	if self == other {
		return 0
	} else if self > other {
		return 1
	} else {
		return -1
	}
}

// UnmarshalText parses string data into a version object
func (v *Version) UnmarshalText(data []byte) (err error) {
	m := pattern.FindStringSubmatch(string(data))
	if len(m) == 0 {
		return fmt.Errorf("invalid version format")
	}

	if major, err := strconv.ParseUint(m[1], 10, 16); err != nil {
		return err
	} else {
		v.major = uint16(major)
	}

	if minor, err := strconv.ParseUint(m[2], 10, 16); err != nil {
		return err
	} else {
		v.minor = uint16(minor)
	}

	if build, err := strconv.ParseUint(m[3], 10, 16); err != nil {
		return err
	} else {
		v.build = uint16(build)
	}

	v.tag = m[4]

	return nil
}

// New creates a new version
func New(major, minor, build uint16, tag string) Version {
	return Version{major, minor, build, tag}
}

// Parse a version object from string
func Parse(v string) (version Version, err error) {
	err = version.UnmarshalText([]byte(v))
	return
}

// Range defines a version range. Usually used for
// version checks
type Range [2]Version

// NewRange creates a new range
func NewRange(from, to Version) Range {
	return [2]Version{from, to}
}

// Has checks if version range has this version
func (r Range) Has(version Version) bool {
	return r[0].Compare(version) <= 0 && r[1].Compare(version) >= 0
}
