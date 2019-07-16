package schema

import (
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
		Name:     "something",
		Indexed:  true,
		TypeText: `"hello world" (S)`,
	}, prop); !ok {
		t.Error()
	}

	prop, err = property("wrong hello world # comment")

	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestTypeDef(t *testing.T) {
	typ, err := typeDef(`"hello" (S)`)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Type{
		Default: `"hello"`,
		Type:    "S",
	}, typ); !ok {
		t.Error()
	}

}
