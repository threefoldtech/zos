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

func (p *PersonResource) SetAge(ctx Context, age uint) (void Void, err error) {
	fmt.Println("setting age:", age)
	person, err := p.Current(ctx)
	if err != nil {
		return void, err
	}

	person.Age = age
	return void, p.Set(ctx, person)
}

// tests normal operation of a resource
func TestResource(t *testing.T) {
	var ptyp PersonResource

	store := NewMemStore()
	store.SpaceCreate(0, "space")

	ctx := &engineContext{
		ctx:    context.TODO(),
		space:  "space",
		user:   0,
		object: "person1",
		exists: false,
		typ:    "Person",
		engine: NewEngine(store),
	}

	resource := NewResourceBuilder[Person]().
		WithAction("create", NewAction(ptyp.Create), ServiceObjectMustNotExist).
		WithAction("set-age", NewAction(ptyp.SetAge)).
		Build()

	response, err := resource.call(
		ctx,
		ResourceRequest{
			Action:  "create",
			Payload: json.RawMessage(`"azmy"`),
		},
	)

	require.NoError(t, err)
	require.EqualValues(t, "null", response.Payload)

	record := store.users[0].spaces["space"].objects["person1"]
	require.EqualValues(t, "Person", record.Type)
	require.EqualValues(t, "person1", record.ID)

	var p Person
	require.NoError(t, json.Unmarshal(record.Data, &p))

	require.EqualValues(t, Person{Name: "azmy"}, p)

	// trying create again should fail since the object already exists
	// this is set by the engine but we can now
	// set it manually here.
	ctx.exists = true
	_, err = resource.call(
		ctx,
		ResourceRequest{
			Action:  "create",
			Payload: json.RawMessage(`"azmy"`),
		},
	)

	require.ErrorIs(t, ErrObjectExists, err)

	_, err = resource.call(
		ctx,
		ResourceRequest{
			Action:  "set-age",
			Payload: json.RawMessage(`40`),
		},
	)

	require.NoError(t, err)

	record = store.users[0].spaces["space"].objects["person1"]
	require.NoError(t, json.Unmarshal(record.Data, &p))

	require.EqualValues(t, Person{Name: "azmy", Age: 40}, p)
}
