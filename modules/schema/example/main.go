package main

import (
	"os"
	"strings"

	"github.com/threefoldtech/zosv2/modules/schema"
)

func main() {
	const input = `
@url =  jumpscale.digitalme.package
name = "UNKNOWN" (S)    #official name of the package, there can be no overlap (can be dot notation)
enable = true (B)
numerics = (LN)
args = (LO) !jumpscale.digitalme.package.arg
loaders= (LO) !jumpscale.digitalme.package.loader

@url =  jumpscale.digitalme.package.arg
key = "" (S)
val =  "" (S)

@url =  jumpscale.digitalme.package.loader
giturl =  (S)
dest =  (S)
enable = true (B)
creation = (D)
	`

	schema, err := schema.New(strings.NewReader(input))

	if err := schema.GenerateGolang(os.Stdout, "test", schema); err != nil {
		panic(err)
	}

	//TODO: validate generated structures!
}
