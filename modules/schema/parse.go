package schema

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	// URL directive
	URL = "url"
)

var (
	directiveRe = regexp.MustCompile(`@(\w+)\s*=\s*([^\#]+)`)
	propertyRe  = regexp.MustCompile(`(\w+)(\*)?\s*=\s*([^\#]+)`)
	typeRe      = regexp.MustCompile(`([^\(]*)\(([^\)]+)\)`)
)

// Schema is a container for all objects defined in the source
type Schema struct {
	Objects []*Object
}

// Object defines a schema object
type Object struct {
	Directives []Directive
	Properties []Property
}

// New reads the schema and return schema description objects
func New(r io.Reader) (schema Schema, err error) {
	scanner := bufio.NewScanner(r)
	var current *Object
	nr := 0
	for scanner.Scan() {
		nr++
		line := strings.TrimSpace(scanner.Text())

		if len(line) == 0 {
			continue
		}

		switch line[0] {
		case '#':
			//comment line, we can skip it
			continue
		case '@':
			//directive line, need to be processed and acted upon
			dir, err := directive(line)
			if err != nil {
				return schema, err
			}

			if dir.Key == URL {
				//push current
				current = new(Object)
			} else if current == nil {
				//directive out of scoop!
				return schema, fmt.Errorf("unexpected directive [line %d]: %s", nr, dir)
			} else {
				current.Directives = append(current.Directives, dir)
			}
		default:
			if current == nil {
				return schema, fmt.Errorf("unexpected token [line %d]: %s", nr, line)
			}

			prop, err := property(line)
			if err != nil {
				return schema, fmt.Errorf("failed to parse property [line %d]: %s", nr, err)
			}

			current.Properties = append(current.Properties, prop)
		}
	}

	return
}

// Directive is a piece of information attached to a schema object
type Directive struct {
	Key   string
	Value string
}

func (d *Directive) String() string {
	return fmt.Sprintf("@%s = %s", d.Key, d.Value)
}

func directive(line string) (dir Directive, err error) {
	m := directiveRe.FindStringSubmatch(line)
	if m == nil {
		return dir, fmt.Errorf("invalid directive line")
	}

	return Directive{Key: strings.ToLower(m[1]), Value: strings.TrimSpace(m[2])}, nil
}

// Property defines a schema property
type Property struct {
	Name     string
	Indexed  bool
	TypeText string
}

func property(line string) (property Property, err error) {
	m := propertyRe.FindStringSubmatch(line)
	if m == nil {
		return property, fmt.Errorf("invalid property definition")
	}

	property.Name = m[1]
	property.Indexed = m[2] == "*"
	property.TypeText = strings.TrimSpace(m[3])

	return
}

// Type holds type information for a property
type Type struct {
	Default string
	Type    string
}

func typeDef(def string) (t Type, err error) {
	// typeDef parses a type text and create a solid type definition
	// current default values is ignored!
	// <default> (Type)

	m := typeRe.FindStringSubmatch(def)
	if m == nil {
		return t, fmt.Errorf("invalid type definition")
	}

	t.Default = strings.TrimSpace(m[1])
	t.Type = m[2]

	return t, nil
}
