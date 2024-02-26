package engine

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type Person struct {
	Name string
	Age  uint
	Kids []string
}

// type definition
type PersonResource struct {
	BaseResource[Person]
}

// the action must be a valid action function
func (p *PersonResource) Create(ctx Context, name string) (void Void, err error) {
	fmt.Println("creating user")

	if err := p.Set(ctx, Person{Name: name}); err != nil {
		return void, err
	}

	return void, nil
}

func TestTypeBuilder(t *testing.T) {
	var ptyp PersonResource

	resource := NewResourceBuilder[Person](false).
		Action("create", NewAction(ptyp.Create)).
		IntoResource()

	response, err := resource.call(
		&engineContext{},
		ResourceRequest{
			Action:  "create",
			Payload: json.RawMessage(`"azmy"`),
		},
	)

	require.NoError(t, err)
	fmt.Println(response.Payload)
}
