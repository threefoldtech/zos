package vm

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/ulikunitz/xz"
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

func gUnzip(data []byte) (reader io.Reader, err error) {
	headerBytes := []byte("\037\213\010") // []byte{0x1f, 0x8b, 8} -> [31, 139, 8]

	var headerIndex int
	var r *gzip.Reader

	for i := 0; i < bytes.Count(data, headerBytes); i++ {
		headerIndex += bytes.Index(data[headerIndex:], headerBytes)
		fmt.Printf("headerIndex: %v\n", headerIndex)

		r, err = gzip.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return
		}
		defer r.Close()

		headerIndex += len(headerBytes)
	}

	reader = r
	return
}

func unXZ(data []byte) (reader io.Reader, err error) {
	headerBytes := []byte("\3757zXZ\000") // [253 55 122 88 90 0]

	var headerIndex int

	for i := 0; i < bytes.Count(data, headerBytes); i++ {
		headerIndex += bytes.Index(data[headerIndex:], headerBytes)
		fmt.Printf("headerIndex: %v\n", headerIndex)

		reader, err = xz.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return
		}

		headerIndex += len(headerBytes)
	}

	return
}
