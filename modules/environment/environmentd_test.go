package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/zosv2/modules/kernel"
)

// There are no test against GetEnvironment since the
// result cannot be deterministic if you have kernel
// argument set or not
func TestManager(t *testing.T) {
	// development mode
	params := kernel.Params{"runmode": {"development"}}
	value := getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "development")

	// testing mode
	params = kernel.Params{"runmode": {"testing"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "testing")

	// main mode
	params = kernel.Params{"runmode": {"production"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "production")

	// fallback
	params = kernel.Params{"nope": {"lulz"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "development")

	// fallback on undefined
	params = kernel.Params{"runmode": {"dunno"}}
	value = getEnvironmentFromParams(params)

	assert.Equal(t, value.runningMode, "development")

}
