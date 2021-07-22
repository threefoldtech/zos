package gridtypes

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	nameMatch     = regexp.MustCompile("^[a-zA-Z0-9_]+$")
	reservedNames = map[Name]struct{}{
		"ygg": {},
	}
)

// DeploymentID is a global unique id for a deployment
type DeploymentID string

// ToPath drive a filepath from the ID
func (i DeploymentID) ToPath() string {
	if len(i) == 0 {
		panic("id is not set")
	}

	return strings.Replace(string(i), "-", "/", -1)
}

// Parts split id into building parts
func (i DeploymentID) Parts() (twin, deployment uint32, err error) {
	_, err = fmt.Sscanf(string(i), "%d-%d", &twin, &deployment)
	return
}

// WorkloadID is a global unique id for a workload
type WorkloadID string

// ToPath drive a filepath from the ID
func (i WorkloadID) ToPath() string {
	if len(i) == 0 {
		panic("id is not set")
	}

	return strings.Replace(string(i), "-", "/", -1)
}

func (i WorkloadID) String() string {
	return string(i)
}

// Parts split id into building parts
func (i WorkloadID) Parts() (twin uint32, deployment uint64, name string, err error) {
	_, err = fmt.Sscanf(string(i), "%d-%d-%s", &twin, &deployment, &name)
	return
}

// IsValidName validates workload name
func IsValidName(n Name) error {
	if len(n) == 0 {
		return fmt.Errorf("name cannot be empty")
	}

	if !nameMatch.MatchString(string(n)) {
		return fmt.Errorf("unsupported character in workload name")
	}

	if _, ok := reservedNames[n]; ok {
		return fmt.Errorf("invalid name '%s' is reserved", n)
	}
	return nil
}

// NewWorkloadID creates a new global ID from it's parts
func NewWorkloadID(twin uint32, deployment uint64, name Name) (WorkloadID, error) {
	if err := IsValidName(name); err != nil {
		return "", err
	}

	return WorkloadID(fmt.Sprintf("%d-%d-%s", twin, deployment, name)), nil
}
