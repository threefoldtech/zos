package vm

import "os/exec"

func isValidELFKernel(KernelImagePath string) error {
	_, err := exec.Command("readelf", "-h", KernelImagePath).CombinedOutput()
	return err
}
