package vm

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cyberdelia/lzo"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/rs/zerolog/log"
	"github.com/ulikunitz/xz"
	"github.com/ulikunitz/xz/lzma"
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

func bUnzip2(data []byte) (reader io.Reader, err error) {
	headerBytes := []byte("BZh") // [66 90 104]

	var headerIndex int

	for i := 0; i < bytes.Count(data, headerBytes); i++ {
		headerIndex += bytes.Index(data[headerIndex:], headerBytes)
		fmt.Printf("headerIndex: %v\n", headerIndex)

		reader = bzip2.NewReader(bytes.NewBuffer(data))

		headerIndex += len(headerBytes)
	}

	return
}

func unlzma(data []byte) (reader io.Reader, err error) {
	headerBytes := []byte("\135\\0\\0\\0")

	var headerIndex int

	for i := 0; i < bytes.Count(data, headerBytes); i++ {
		headerIndex += bytes.Index(data[headerIndex:], headerBytes)
		fmt.Printf("headerIndex: %v\n", headerIndex)

		reader, err = lzma.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return
		}

		headerIndex += len(headerBytes)
	}

	return
}

func lZop(data []byte) (reader io.Reader, err error) {
	headerBytes := []byte("\211\114\132")

	var headerIndex int
	var r *lzo.Reader

	for i := 0; i < bytes.Count(data, headerBytes); i++ {
		headerIndex += bytes.Index(data[headerIndex:], headerBytes)
		fmt.Printf("headerIndex: %v\n", headerIndex)

		r, err = lzo.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return
		}
		defer r.Close()

		headerIndex += len(headerBytes)
	}

	reader = r
	return
}

func lZ4(data []byte) (reader io.Reader, err error) {
	headerBytes := []byte("\002!L\030")

	var headerIndex int
	var r *lzo.Reader

	for i := 0; i < bytes.Count(data, headerBytes); i++ {
		headerIndex += bytes.Index(data[headerIndex:], headerBytes)
		fmt.Printf("headerIndex: %v\n", headerIndex)

		reader = lz4.NewReader(bytes.NewBuffer(data))

		headerIndex += len(headerBytes)
	}

	reader = r
	return
}

func unZstd(data []byte) (reader io.Reader, err error) {
	headerBytes := []byte("(\265/\375")

	var headerIndex int
	var r *zstd.Decoder

	for i := 0; i < bytes.Count(data, headerBytes); i++ {
		headerIndex += bytes.Index(data[headerIndex:], headerBytes)
		fmt.Printf("headerIndex: %v\n", headerIndex)

		r, err = zstd.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return
		}
		defer r.Close()

		headerIndex += len(headerBytes)
	}

	reader = r
	return
}

func tryDecompressKernel(KernelImagePath string) error {
	if len(strings.TrimSpace(KernelImagePath)) == 0 {
		return fmt.Errorf("kernel image is required")
	}

	// Prepare temp files:
	tmp, err := os.MkdirTemp("/tmp/", "vmlinux-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	kernelData, err := os.ReadFile(KernelImagePath)
	if err != nil {
		return err
	}

	// 1. gUnzip
	reader, err := gUnzip(kernelData)
	if err != nil {
		log.Error().Err(err).Send()
	}

	// 2. unxz
	reader, err = unXZ(kernelData)
	if err != nil {
		log.Error().Err(err).Send()
	}

	// 3. bUnzip2
	reader, err = bUnzip2(kernelData)
	if err != nil {
		log.Error().Err(err).Send()
	}

	// 4. unlzma
	reader, err = unlzma(kernelData)
	if err != nil {
		log.Error().Err(err).Send()
	}

	// 5. lzop
	reader, err = lZop(kernelData)
	if err != nil {
		log.Error().Err(err).Send()
	}

	// 6. lz4
	reader, err = lZ4(kernelData)
	if err != nil {
		log.Error().Err(err).Send()
	}

	// 7. unzstd
	reader, err = unZstd(kernelData)
	if err != nil {
		log.Error().Err(err).Send()
	}

	// TODO: handle err

	return writer(reader, fmt.Sprintf("%s/%s", tmp, filepath.Base(KernelImagePath)))
}
