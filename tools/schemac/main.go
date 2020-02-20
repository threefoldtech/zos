package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/threefoldtech/zos/pkg/schema"
)

func handle(file, pkg, dir string) error {
	fd, err := os.Open(file)
	if err != nil {
		return err
	}

	defer fd.Close()

	sch, err := schema.New(fd)
	if err != nil {
		return err
	}
	out := strings.TrimSuffix(path.Base(fd.Name()), path.Ext(fd.Name()))
	w, err := os.Create(path.Join(dir, fmt.Sprintf("%s.go", out)))
	if err != nil {
		return err
	}

	return schema.GenerateGolang(w, pkg, sch)
}

func main() {
	var (
		pkg string
		dir string
		in  string
	)

	flag.StringVar(&pkg, "pkg", "schema", "package name in generated go code")
	flag.StringVar(&dir, "dir", ".", "directory to output files to")
	flag.StringVar(&in, "in", "", "directory with schema files. process all files in the directory. Otherwise process all input files given as extra args")

	flag.Parse()
	files := flag.Args()
	if len(in) != 0 {
		fs, err := ioutil.ReadDir(in)
		if err != nil {
			log.Fatalf("failed to list files in director(%s): %s", in, err)
		}
		for _, f := range fs {
			if f.IsDir() {
				continue
			}
			if !strings.HasSuffix(f.Name(), ".toml") {
				continue
			}

			files = append(files, filepath.Join(in, f.Name()))
		}
	}

	for _, input := range files {
		if err := handle(input, pkg, dir); err != nil {
			log.Fatalf("error while generating schema: %s", err)
		}
	}
}
