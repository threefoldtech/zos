package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr/v2"
	"github.com/gregdhill/go-openrpc/generate"
	"github.com/gregdhill/go-openrpc/parse"
	"github.com/gregdhill/go-openrpc/types"
)

var specFile = "openrpc.json"
var rpcPkgDir = "../../pkg/zos_rpc"

func main() {
	if err := os.Remove(filepath.Join(rpcPkgDir, "/server.go")); err != nil {
		log.Fatal(err)
	}
	if err := os.Remove(filepath.Join(rpcPkgDir, "/types.go")); err != nil {
		log.Fatal(err)
	}

	data, err := os.ReadFile(specFile)
	if err != nil {
		log.Fatal(err)
	}

	spec := types.NewOpenRPCSpec1()
	if err = json.Unmarshal(data, spec); err != nil {
		log.Fatal(err)
	}

	parse.GetTypes(spec, spec.Objects)
	box := packr.New("template", "./templates")

	if err = generate.WriteFile(box, "server", rpcPkgDir, spec); err != nil {
		log.Fatal(err)
	}

	if err = generate.WriteFile(box, "types", rpcPkgDir, spec); err != nil {
		log.Fatal(err)
	}

}
