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
	major int64
	minor int64
	build int64
	tag   string
}

// String formats version as 'v<Major>.<Minor>.<Build><Tag>'
func (v *Version) String() string {
	return fmt.Sprintf("v%d.%d.%d%s", v.major, v.minor, v.build, v.tag)
}

// MarshalText convert version object to text
func (v Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText parses string data into a version object
func (v *Version) UnmarshalText(data []byte) (err error) {
	m := pattern.FindStringSubmatch(string(data))
	if len(m) == 0 {
		return fmt.Errorf("invalid version format")
	}

	if v.major, err = strconv.ParseInt(m[1], 10, 64); err != nil {
		return
	}

	if v.minor, err = strconv.ParseInt(m[2], 10, 64); err != nil {
		return
	}

	if v.build, err = strconv.ParseInt(m[3], 10, 64); err != nil {
		return
	}

	v.tag = m[4]

	return nil
}

// New creates a new version
func New(major, minor, build int64, tag string) Version {
	return Version{major, minor, build, tag}
}

// Parse a version object from string
func Parse(v string) (version Version, err error) {
	err = version.UnmarshalText([]byte(v))
	return
}
