package vm

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cyberdelia/lzo"
	"github.com/hashicorp/go-multierror"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/rs/zerolog/log"
	"github.com/ulikunitz/xz"
	"github.com/ulikunitz/xz/lzma"
)

// TODO: sudo apt install liblzo2-dev

func isValidELFKernel(KernelImagePath string) error {
	f, err := elf.Open(KernelImagePath)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

type algoOptions struct {
	decompressFunc func(kernelStream io.Reader) (io.Reader, error)
	headerBytes    []byte
}

// TODO: []byte -> io.reader
func decompressData(data []byte, tmpFile *os.File, o algoOptions) error {
	var headerIndex int
	var errs error

	for i := 0; i < bytes.Count(data, o.headerBytes); i++ {
		headerIndex += bytes.Index(data[headerIndex:], o.headerBytes)

		r, err := o.decompressFunc(bytes.NewBuffer(data[headerIndex:]))
		if err != nil {
			return err
		}

		_, err = io.Copy(tmpFile, r)
		if err != nil {
			return err
		}

		err = isValidELFKernel(tmpFile.Name())
		if err == nil {
			return nil
		}

		errs = multierror.Append(errs, err)
		headerIndex += len(o.headerBytes)
	}

	return errs
}

func gUnzip(kernelStream io.Reader) (io.Reader, error) {
	r, err := gzip.NewReader(kernelStream)
	if err != nil {
		return nil, err
	}

	r.Multistream(false)
	// TODO: how it is read after closing !!!!
	defer r.Close()

	return r, nil
}

func unXZ(kernelStream io.Reader) (io.Reader, error) {
	return xz.NewReader(kernelStream)
}

func bUnzip2(kernelStream io.Reader) (io.Reader, error) {
	return bzip2.NewReader(kernelStream), nil
}

func unlzma(kernelStream io.Reader) (io.Reader, error) {
	return lzma.NewReader(kernelStream)
}

func lZop(kernelStream io.Reader) (io.Reader, error) {
	r, err := lzo.NewReader(kernelStream)
	r.Close()
	return r, err
}

func lZ4(kernelStream io.Reader) (io.Reader, error) {
	return lz4.NewReader(kernelStream), nil
}

func unZstd(kernelStream io.Reader) (io.Reader, error) {
	r, err := zstd.NewReader(kernelStream)
	r.Close()
	return r, err
}

func tryDecompressKernel(KernelImagePath string) error {
	if len(strings.TrimSpace(KernelImagePath)) == 0 {
		return fmt.Errorf("kernel image is required")
	}

	// if kernel is already an elf (uncompressed)
	if err := isValidELFKernel(KernelImagePath); err == nil {
		return nil
	}

	// Prepare temp files:
	tmpFile, err := os.CreateTemp("/tmp", "vmlinux-")
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	kernelData, err := os.ReadFile(KernelImagePath)
	if err != nil {
		return err
	}

	algos := []algoOptions{
		{
			decompressFunc: gUnzip,
			headerBytes:    []byte("\037\213\010"),
		},
		{
			decompressFunc: unXZ,
			headerBytes:    []byte("\3757zXZ\000"),
		},
		{
			decompressFunc: bUnzip2,
			headerBytes:    []byte("BZh"),
		},
		{
			decompressFunc: unlzma,
			headerBytes:    []byte("\135\\0\\0\\0"),
		},
		{
			decompressFunc: lZop,
			headerBytes:    []byte("\211\114\132"),
		},
		{
			decompressFunc: lZ4,
			headerBytes:    []byte("\002!L\030"),
		},
		{
			decompressFunc: unZstd,
			headerBytes:    []byte("(\265/\375"),
		},
	}

	// 1. gUnzip
	for _, algo := range algos {
		err = decompressData(kernelData, tmpFile, algo)
		if err == nil {
			break
		}
		log.Error().Err(err).Send()
	}

	return nil
}
