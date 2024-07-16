package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"

	"github.com/threefoldtech/zos/tools/openrpc-codegen/fileutils"
	"github.com/threefoldtech/zos/tools/openrpc-codegen/generator"
)

type flags struct {
	spec   string
	output string
	pkg    string
}

func run() error {
	var f flags
	flag.StringVar(&f.spec, "spec", "", "openrpc spec file")
	flag.StringVar(&f.output, "output", "", "generated go code")
	flag.StringVar(&f.pkg, "pkg", "apirpc", "name of the go package")
	flag.Parse()

	if f.spec == "" || f.output == "" {
		return fmt.Errorf("missing flag is required")
	}

	spec, err := fileutils.Parse(f.spec)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := generator.GenerateServerCode(&buf, spec, f.pkg); err != nil {
		return err
	}

	if err := fileutils.Write(buf, f.output); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
