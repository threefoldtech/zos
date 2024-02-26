package engine

import (
	"context"
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
type PersonType struct {
	BaseObject[Person]
}

// the action must be a valid action function
func (p *PersonType) Create(ctx context.Context, name string) (void Void, err error) {
	fmt.Println("creating user")

	if err := p.Set(ctx, Person{Name: name}); err != nil {
		return void, err
	}

	return void, nil
}

func TestTypeBuilder(t *testing.T) {
	var ptyp PersonType
	typ := NewTypeBuilder[Person](false).
		Action("create", ActionFn[string, Void](ptyp.Create).IntoService()).
		IntoResource()

	response, err := typ.Do(
		context.Background(),
		ObjectRequest{
			Action:  "create",
			Payload: json.RawMessage(`"azmy"`),
		},
	)

	require.NoError(t, err)
	fmt.Println(response.Payload)
}
