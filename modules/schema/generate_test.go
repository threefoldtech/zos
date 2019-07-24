package schema

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateGO(t *testing.T) {
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

	schema, err := New(strings.NewReader(input))

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	var buf strings.Builder

	if err := GenerateGolang(&buf, "test", schema); err != nil {
		t.Fatal(err)
	}

	fmt.Println(buf.String())

	//TODO: validate generated structures!
}
