package tpm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func tpm(ctx context.Context, name string, out interface{}, arg ...string) error {
	name = fmt.Sprintf("tpm2_%s", name)

	cmd := exec.CommandContext(ctx, name, arg...)

	output, err := cmd.Output()
	if err, ok := err.(*exec.ExitError); ok && err != nil {
		return errors.Wrapf(err, "error while running command: (%s)", string(err.Stderr))
	} else if err != nil {
		return errors.Wrap(err, "failed to run tpm")
	}

	if out == nil {
		return nil
	}

	decoder := yaml.NewDecoder(bytes.NewBuffer(output))

	return decoder.Decode(out)
}

// IsTPMSupported checks if TPM is accessible on this system
func IsTPMSupported(ctx context.Context) bool {
	pcrs, err := PCRs(ctx)
	if err != nil {
		return false
	}

	return len(pcrs) > 0
}

// PersistedHandlers return a list of persisted handlers on the system
func PersistedHandlers(ctx context.Context) (handlers []string, err error) {
	err = tpm(ctx, "getcap", &handlers, "handles-persistent")
	return
}

// PCRs returns the available PCRs numbers as map of [hash-algorithm][]int
func PCRs(ctx context.Context) (map[string][]int, error) {
	var data struct {
		Top []map[string][]int `yaml:"selected-pcrs"`
	}

	if err := tpm(ctx, "getcap", &data, "pcrs"); err != nil {
		return nil, err
	}

	pcrs := make(map[string][]int)
	for _, m := range data.Top {
		for k, l := range m {
			pcrs[k] = l
		}
	}

	return pcrs, nil
}
