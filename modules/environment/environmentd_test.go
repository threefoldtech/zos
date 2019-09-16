package environment

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zosv2/modules/kernel"
)

// There are no test against GetEnvironment since the
// result cannot be deterministic if you have kernel
// argument set or not
func TestManager(t *testing.T) {
	// Development mode
	params := kernel.Params{"runmode": {"development"}}
	value := getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "development")

	// Testing mode
	params = kernel.Params{"runmode": {"testing"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "testing")

	// Main mode
	params = kernel.Params{"runmode": {"production"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "production")

	// Fallback
	params = kernel.Params{"nope": {"lulz"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "development")

	// Fallback on undefined
	params = kernel.Params{"runmode": {"dunno"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "development")
}

func TestEnvironmentOverride(t *testing.T) {
	os.Setenv("ZOS_BCDB_URL", "localhost:1234")

	params := kernel.Params{"runmode": {"development"}}
	value := getEnvironmentFromParams(params)

	assert.Equal(t, value.bcdbUrl, "localhost:1234")
}
