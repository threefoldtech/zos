package schema

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirective(t *testing.T) {
	dir, err := directive("@url = hello world # comment")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Directive{Key: "url", Value: "hello world"}, dir); !ok {
		t.Error()
	}

	dir, err = directive("@url hello world # comment")

	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestProperty(t *testing.T) {
	prop, err := property(`something* = "hello world" (S) # comment`)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Property{
		Name:    "something",
		Indexed: true,
		Type: Type{
			Default: `"hello world"`,
			Kind:    StringKind,
		},
	}, prop); !ok {
		t.Error()
	}

	prop, err = property("wrong hello world # comment")

	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestTypeDef(t *testing.T) {

	cases := []struct {
		Input  string
		Output *Type
	}{
		{
			`"hello world" (S)`,
			&Type{
				Default: `"hello world"`,
				Kind:    StringKind,
			},
		},
		{
			`(LO) ! refernece.to.another.object`,
			&Type{
				Kind: ListKind,
				Element: &Type{
					Kind:      ObjectKind,
					Reference: "refernece.to.another.object",
				},
			},
		},
		{
			`(Lmultiline) ! refernece.to.another.object`,
			&Type{
				Kind: ListKind,
				Element: &Type{
					Kind: MultilineKind,
				},
			},
		},
		{
			`"1,2,3" (LI)`,
			&Type{
				Kind:    ListKind,
				Default: `"1,2,3"`,
				Element: &Type{
					Kind: IntegerKind,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Input, func(t *testing.T) {
			typ, err := typeDef(c.Input)

			if ok := assert.NoError(t, err); !ok {
				t.Fatal()
			}

			if ok := assert.EqualValues(t, c.Output, typ); !ok {
				t.Error()
			}
		})
	}

}

func TestTokenScan(t *testing.T) {
	cases := []struct {
		Input  string
		Output []string
	}{
		{"I", []string{"I"}},
		{"II", []string{"I", "I"}},
		{"Lstring", []string{"L", "string"}},
		{"LstringI", []string{"L", "string", "I"}},
		{"LO", []string{"L", "O"}},
		{"", nil},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("case.%s", c.Input), func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(c.Input))
			scanner.Split(tokenSplit)

			var out []string
			for scanner.Scan() {
				out = append(out, scanner.Text())
			}

			if ok := assert.Equal(t, c.Output, out); !ok {
				t.Error()
			}
		})
	}
}

func TestSimpleNew(t *testing.T) {
	input := `
# this is a comment

@url =  jumpscale.digitalme.package
name = "UNKNOWN" (S)    #official name of the package, there can be no overlap (can be dot notation)
enable = true (B)
args = (LO) !jumpscale.digitalme.package.arg
loaders= (LO) !jumpscale.digitalme.package.loader
	`

	schema, err := New(strings.NewReader(input))

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, schema, 1); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, &Object{
		URL: "jumpscale.digitalme.package",
		Properties: []Property{
			Property{Name: "name", Type: Type{
				Default: `"UNKNOWN"`,
				Kind:    StringKind,
			}},
			Property{Name: "enable", Type: Type{
				Default: `true`,
				Kind:    BoolKind,
			}},
			Property{Name: "args", Type: Type{
				Kind: ListKind,
				Element: &Type{
					Kind:      ObjectKind,
					Reference: "jumpscale.digitalme.package.arg",
				},
			}},
			Property{Name: "loaders", Type: Type{
				Kind: ListKind,
				Element: &Type{
					Kind:      ObjectKind,
					Reference: "jumpscale.digitalme.package.loader",
				},
			}},
		},
	}, schema[0]); !ok {
		t.Error()
	}
}
