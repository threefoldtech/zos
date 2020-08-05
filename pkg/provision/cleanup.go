package provision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
	"golang.org/x/net/context"
)

func execute(cmd string, args ...string) ([]byte, error) {
	exe := exec.Command(cmd, args...)
	output, err := exe.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "execute '%v %+v': %v", cmd, args, string(output))
	}

	return output, nil
}

func lines(input []byte) []string {
	input = bytes.TrimSpace(input)
	if len(input) == 0 {
		return nil
	}

	return strings.Split(string(input), "\n")
}

func must(input []byte, err error) []byte {
	if err != nil {
		panic(err)
	}

	return input
}

type container struct {
	Spec struct {
		Root struct {
			Path string `json:"path"`
		} `json:"root"`
		Mounts []struct {
			Source string `json:"source"`
		} `json:"mounts"`
	} `json:"Spec"`
}

// CleanupResources cleans up unused resources
func CleanupResources() error {
	toSave := make(map[string]struct{})

	for _, ns := range lines(must(execute("ctr", "namespace", "ls", "-q"))) {
		for _, cnt := range lines(must(execute("ctr", "-n", ns, "container", "ls", "-q"))) {
			bytes := must(execute("ctr", "-n", ns, "container", "info", cnt))
			var container container
			if err := json.Unmarshal(bytes, &container); err != nil {
				panic(err)
			}

			toSave[filepath.Base(container.Spec.Root.Path)] = struct{}{}
			for _, mnt := range container.Spec.Mounts {
				if strings.HasPrefix(mnt.Source, "/mnt/") {
					toSave[filepath.Base(mnt.Source)] = struct{}{}
				}
			}
		}
	}

	ctx := context.TODO()
	utils := filesystem.NewUtils()
	pools, err := ioutil.ReadDir("/mnt")
	if err != nil {
		return errors.Wrap(err, "failed to list available pools")
	}

	for _, pool := range pools {
		if !pool.IsDir() {
			continue
		}
		path := filepath.Join("/mnt", pool.Name())
		subvols, err := utils.SubvolumeList(ctx, path)
		if err != nil {
			return errors.Wrapf(err, "failed to list available subvolumes in: %s", path)
		}

		qgroups, err := utils.QGroupList(ctx, path)
		if err != nil {
			return errors.Wrapf(err, "failed to list available cgroups in: %s", path)
		}
		for _, subvol := range subvols {
			qgroup, ok := qgroups[fmt.Sprintf("0/%d", subvol.ID)]
			if !ok {
				log.Info().Msgf("skipping volume '%s' has no assigned quota", subvol.Path)
				continue
			}

			// we only handle volumes that are 256MiB or 10MiB
			if qgroup.MaxRfer != 268435456 && qgroup.MaxRfer != 10485760 {
				log.Info().Msgf("skipping volume '%s' is of size: %d", subvol.Path, qgroup.MaxRfer)
				continue
			}

			// now, is this subvol in one of the toSave ?
			if _, ok := toSave[filepath.Base(subvol.Path)]; ok {
				log.Info().Msgf("skipping volume '%s' is used", subvol.Path)
				continue
			}

			// delete the subvolume
			log.Info().Msgf("deleting subvolume %s", subvol.Path)
			if err := utils.SubvolumeRemove(ctx, filepath.Join(path, subvol.Path)); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", subvol.Path)
			}
			if err := utils.QGroupDestroy(ctx, qgroup.ID, path); err != nil {
				log.Err(err).Msgf("failed to delete qgroup: '%s'", qgroup.ID)
			}
		}
	}

	return nil
}
