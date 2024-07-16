package generator

type structType struct {
	Name   string
	Fields []field
}

type field struct {
	Name     string
	Type     string
	JSONName string
}

const structTemplate = `
type {{.Name}} struct {
{{- range .Fields }}
  {{ .Name }} {{ .Type }} ` + "`json:\"{{ .JSONName }}\"`" + `
{{- end }}
}
`

type service struct {
	Name    string
	Methods []methodType
}

type methodType struct {
	Name      string
	ArgType   string
	ReplyType string
}

const MethodTemplate = `
type {{.Name}} interface {
{{- range .Methods }}
  {{.Name}}({{.ArgType}}, *{{.ReplyType}}) error
{{- end}}
}
`
