package tpm

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type HexString string

func (h HexString) Bytes() ([]byte, error) {
	return hex.DecodeString(string(h))
}

type HashKind string

const (
	SHA1   HashKind = "sha1"
	SHA256 HashKind = "sha256"
	SHA384 HashKind = "sha384"
	SHA512 HashKind = "sha512"
)

type PCRSelector map[HashKind][]int

func (p PCRSelector) String() string {
	// to make it consistent we need to
	// sort the the map keys first
	var keys []HashKind
	for hash := range p {
		keys = append(keys, hash)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var buf strings.Builder
	for _, hash := range keys {
		if buf.Len() > 0 {
			buf.WriteRune('+')
		}
		buf.WriteString(string(hash))
		buf.WriteRune(':')
		for i, id := range p[hash] {
			if i != 0 {
				buf.WriteRune(',')
			}
			buf.WriteString(fmt.Sprint(id))
		}
	}

	return buf.String()
}

// File is a tmp file path to make it easier to pass files around
type File string

func (f File) Delete() error {
	return os.Remove(string(f))
}

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

func PCRPolicy(ctx context.Context, selector PCRSelector) (out HexString, err error) {
	err = tpm(ctx, "createpolicy", &out, "--policy-pcr", "-l", selector.String(), "-L", "/dev/null")
	return
}
