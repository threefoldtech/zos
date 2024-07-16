package generator

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"text/template"
)

func convertToGoType(t string, f string) string {
	if f == "raw" {
		return "json.RawMessage"
	}

	switch t {
	case "integer":
		return "uint64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "null":
		return "any"
	default:
		return t
	}
}

func invokeRef(t string) string {
	return path.Base(t)
}

func extractMethodName(methodText string) string {
	return strings.Split(methodText, ".")[len(strings.Split(methodText, "."))-1]
}

func executeTemplate(buf *bytes.Buffer, tmpl string, data interface{}) error {
	templ, err := template.New("").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	if err := templ.Execute(buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}

func addPackageName(buf *bytes.Buffer, pkg string) error {
	pkgLine := fmt.Sprintf("package %s\n", pkg)
	_, err := buf.Write([]byte(pkgLine))
	return err
}
