package schema

import (
	"github.com/dave/jennifer/jen"
)

// GenerateGolang generate type stubs for Go from schema
func GenerateGolang(schema Schema) {
	jen.NewFile("package")
}
