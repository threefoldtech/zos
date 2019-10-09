package container

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _startup = `[startup]

[startup.entry]
name = "core.system"

[startup.entry.args]
name = "/start"
dir = "/data"

[startup.entry.args.env]
DIFFICULTY = "easy"
LEVEL = "world"
SERVER_PORT = "25565"
`

func TestReadEnvs(t *testing.T) {
	r := strings.NewReader(_startup)

	envs, err := readEnvs(r)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"DIFFICULTY=easy",
		"LEVEL=world",
		"SERVER_PORT=25565",
	}, envs)
}

func TestMergeEnvs(t *testing.T) {
	env := mergeEnvs(
		[]string{"FOO=BAR", "HELLO=WORLD"},
		[]string{"HELLO=HELLO"},
	)

	assert.Equal(t,
		sort.StringSlice([]string{"HELLO=WORLD", "FOO=BAR"}),
		sort.StringSlice(env))
}
