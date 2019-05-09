package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"

	"github.com/threefoldtech/zosv2/modules"
)

const (
	defaultStorage = "zdb://hub.grid.tf:9900"
	defaultRoot    = "/var/modules/flist"
)

type flistModule struct {
	root string // root directory where all the working file of the module will be located
}

func New(root string) *flistModule {
	if root == "" {
		root = defaultRoot
	}

	// prepare directory layout for the module
	for _, path := range []string{"flist", "backend", "pid", "log", "mountpoint"} {
		if err := os.MkdirAll(filepath.Join(root, path), 0770); err != nil {
			panic(err)
		}
	}

	return &flistModule{
		root: root,
	}
}

func (f *flistModule) Mount(url, storage string) (string, error) {
	if storage == "" {
		storage = defaultStorage
	}

	rc, err := downloadFlist(url)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	flistPath, err := f.saveFlist(rc)
	if err != nil {
		return "", err
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	uuidStr := uuid.String()
	mountpoint := filepath.Join(f.root, "mounpoint", uuidStr)
	if err := os.MkdirAll(mountpoint, 0770); err != nil {
		return "", err
	}
	pidPath := filepath.Join(f.root, "pid", uuidStr)
	logPath := filepath.Join(f.root, "log", uuidStr)
	cmd := exec.Command("g8ufs",
		"-backend",
		filepath.Join(f.root, "backend"),
		"-meta",
		flistPath,
		"-storage-url",
		storage,
		"-daemon",
		"-pid",
		pidPath,
		"-logfile",
		logPath,
		mountpoint)

	if err := cmd.Run(); err != nil {
		return "", err
	}

	_, err = os.Stat(pidPath)
	if err != nil {
		return "", err
	}

	return mountpoint, nil
}
func (f *flistModule) Umount(path string) error {
	return syscall.Unmount(path, syscall.MNT_DETACH)
}

func downloadFlist(url string) (io.ReadCloser, error) {
	// todo check if we don't already have it
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fail to download flist: %v", resp.Status)
	}
	return resp.Body, nil
}

func (f *flistModule) saveFlist(r io.Reader) (string, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", md5.Sum(b))
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	path := filepath.Join(f.root, "flist", id.String(), hash)
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return "", err
	}

	return path, ioutil.WriteFile(path, b, 0440)
}

var _ modules.Flister = (*flistModule)(nil)
