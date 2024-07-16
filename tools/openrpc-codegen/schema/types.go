package schema

type Spec struct {
	OpenRPC    string     `json:"openrpc"`
	Info       Info       `json:"info"`
	Servers    []Server   `json:"servers"`
	Methods    []Method   `json:"methods"`
	Components Components `json:"components"`
}

type Info struct {
	Version string  `json:"version"`
	Title   string  `json:"title"`
	License License `json:"license"`
}

type License struct {
	Name string `json:"name"`
}

type Server struct {
	URL string `json:"url"`
}

type Method struct {
	Name     string    `json:"name"`
	Summary  string    `json:"summary"`
	Tags     []Tag     `json:"tags"`
	Params   []Param   `json:"params"`
	Result   Result    `json:"result"`
	Errors   []Error   `json:"errors"`
	Examples []Example `json:"examples"`
}

type Tag struct {
	Name string `json:"name"`
}

type Param struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Schema      Schema `json:"schema"`
}

type Result struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      Schema `json:"schema"`
}

type Schema struct {
	Type       string            `json:"type"`
	Minimum    int               `json:"minimum,omitempty"`
	Ref        string            `json:"$ref,omitempty"`
	Items      *Schema           `json:"items,omitempty"`
	Required   []string          `json:"required,omitempty"`
	Tag        string            `json:"tag"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Format     string            `json:"format,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Example struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Params      []Param `json:"params"`
	Result      Result  `json:"result"`
}

type Components struct {
	ContentDescriptors map[string]ContentDescriptor `json:"contentDescriptors"`
	Schemas            map[string]Schema            `json:"schemas"`
}

type ContentDescriptor struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Schema      Schema `json:"schema"`
}
