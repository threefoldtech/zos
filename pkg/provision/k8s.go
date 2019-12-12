package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
	"gopkg.in/yaml.v2"
)

// K8sCluster reservation schema
type K8sCluster struct {
	// address of the master node, if empty, then we are the master
	// and need to start k3s as server otherwise stars as agent and connect
	// to this address
	MasterAddr string `json:"master,omitempty"`
	// authentication token
	Token string `json:"token,omitempty"`
}

type K8sClusterResult struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Clusters   []struct {
		Cluster struct {
			CertificateAuthorityData string `json:"certificate-authority-data" yaml:"certificate-authority-data"`
			Server                   string `json:"server" yaml:"server"`
		} `json:"cluster" yaml:"cluster"`
		Name string `json:"name" yaml:"name"`
	} `json:"clusters" yaml:"clusters"`
	Contexts []struct {
		Context struct {
			Cluster   string `json:"cluster" yaml:"cluster"`
			Namespace string `json:"namespace" yaml:"namespace"`
			User      string `json:"user" yaml:"user"`
		} `json:"context" yaml:"context"`
		Name string `json:"name" yaml:"name"`
	} `json:"contexts" yaml:"contexts"`
	CurrentContext string `json:"current- yaml:"currentcontext""`
	Kind           string `json:"kind" yaml:"kind"`
	Preferences    struct {
	} `json:"preferences" yaml:"preferences"`
	Users []struct {
		Name string `json:"name" yaml:"name"`
		User struct {
			Password string `json:"password" yaml:"password"`
			Username string `json:"username" yaml:"username"`
		} `json:"user" yaml:"user"`
	} `json:"users" yaml:"users"`
}

func k8sProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {

	//1. check that there is no "regular" workloads current provisioned.
	//2. provisiond switch to a state where is refused to provision any additional workloads, the node is now fully reserved
	//3. download k3s binaries and prepare directory on cache disk for data directory of k3s
	//4. starts k3s binary in server or agent mode depending on the reservation schema content

	var config K8sCluster
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return ContainerResult{}, err
	}

	identity := stubs.NewIdentityManagerStub(GetZBus(ctx))

	log.Info().Msg("prepare k3s cache storage")
	if err := prepareCache(); err != nil {
		return nil, err
	}

	binPath := filepath.Join(k3sDataDir, "bin", "k3s")
	log.Info().Str("destination", binPath).Msg("download k3s binary")
	if err := downloadK3s(binPath); err != nil {
		return nil, err
	}

	log.Info().Msg("create k3s service and start it")
	cfgPath, err := createZinitService(identity.NodeID().Identity(), binPath, config.MasterAddr, config.Token)
	if err != nil {
		return nil, err
	}

	log.Info().Msg("read k3s configuration file")
	result, err := readK3sConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func k8sDecomission(ctx context.Context, reservation *Reservation) error {

	zinitClient, err := zinit.New("")
	if err != nil {
		return err
	}
	defer zinitClient.Close()

	if err := zinitClient.StopWait(time.Second*10, "k3s"); err != nil {
		return err
	}

	if err := zinitClient.Forget("k3s"); err != nil {
		return err
	}

	if err := zinit.RemoveService("k3s"); err != nil {
		return err
	}

	for _, dst := range dir2Mount {
		if err := syscall.Unmount(dst, 0); err != nil {
			return err
		}
	}

	return os.RemoveAll(k3sDataDir)
}

const k3sDataDir = "/var/cache/k8s"

var dir2Mount = map[string]string{
	filepath.Join(k3sDataDir, "etc"):     "/etc/rancher",
	filepath.Join(k3sDataDir, "kubelet"): "/var/lib/kubelet",
	filepath.Join(k3sDataDir, "rook"):    "/var/lib/rook",
}

func prepareCache() error {

	if err := os.MkdirAll(k3sDataDir, 0770); err != nil {
		return errors.Wrapf(err, "create directory %s", k3sDataDir)
	}

	// directory that needs persistance. we bind mound then onto the cache disk
	for src, dst := range dir2Mount {
		if err := os.MkdirAll(src, 0770); err != nil {
			return errors.Wrapf(err, "create directory %s", src)
		}
		if err := os.MkdirAll(dst, 0770); err != nil {
			return errors.Wrapf(err, "create directory %s", dst)
		}
		if err := syscall.Mount(src, dst, "none", syscall.MS_BIND, ""); err != nil {
			return errors.Wrapf(err, "mount %s on %s", src, dst)
		}
	}

	return nil
}

func downloadK3s(dst string) error {
	const k3sURL = "https://github.com/rancher/k3s/releases/download/v1.0.0/k3s"

	if err := os.MkdirAll(filepath.Dir(dst), 0770); err != nil {
		return err
	}

	resp, err := http.Get(k3sURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0770)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = io.Copy(f, resp.Body); err != nil {
		return err
	}

	return nil
}

func createZinitService(nodeID string, binPath, master, token string) (string, error) {
	zinitClient, err := zinit.New("")
	if err != nil {
		return "", err
	}
	defer zinitClient.Close()

	cfgPath := filepath.Join(k3sDataDir, "k3s.yaml")
	dataDir := filepath.Join(k3sDataDir, "k3s")

	var cmd string
	if master == "" {
		cmd = fmt.Sprintf("%s server --write-kubeconfig %s ", binPath, cfgPath)
	} else {
		cmd = fmt.Sprintf("%s agent --server %s ", binPath, master)
	}

	cmd = fmt.Sprintf("%s --data-dir %s --token %s --node-name %s", cmd, dataDir, token, nodeID)

	err = zinit.AddService("k3s", zinit.InitService{
		Exec:    cmd,
		Oneshot: false,
		Test:    "",
		After:   []string{"networkd"},
		Log:     zinit.RingLogType,
	})
	if err != nil {
		return "", err
	}

	if err := zinitClient.Monitor("k3s"); err != nil {
		return "", err
	}

	return cfgPath, zinitClient.Start("k3s")
}

func readK3sConfig(path string) (K8sClusterResult, error) {

	var result K8sClusterResult
	do := func() error {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(b, &result); err != nil {
			return err
		}

		if len(result.Clusters) <= 0 || result.Clusters[0].Cluster.CertificateAuthorityData == "" {
			return fmt.Errorf("certification not present")
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 5 * time.Second
	bo.MaxElapsedTime = 5 * time.Minute

	if err := backoff.RetryNotify(do, bo, func(err error, t time.Duration) {
		log.Info().Msg("wait for k3s to be started")
		log.Info().Str("result", fmt.Sprintf("%+v", result)).Send()
	}); err != nil {
		return result, err
	}
	return result, nil
}
