package vm

import (
	"io"
	"os"
	"os/exec"
)

func isValidELFKernel(KernelImagePath string) error {
	_, err := exec.Command("readelf", "-h", KernelImagePath).CombinedOutput()
	return err
}

func writer(reader io.Reader, targetPath string) error {
	writer, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return err
}
