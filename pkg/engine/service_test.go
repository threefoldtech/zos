package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAction(t *testing.T) {

	type Input struct {
		X int
		Y int
	}

	type Output struct {
		O int
	}

	add := func(ctx context.Context, input Input) (Output, error) {
		return Output{O: input.X + input.Y}, nil
	}

	multi := func(ctx context.Context, input Input) (Output, error) {
		return Output{O: input.X * input.Y}, nil
	}

	addService := IntoService(ActionFn[Input, Output](add))
	multiService := IntoService(ActionFn[Input, Output](multi))

	added, err := addService.Call(context.TODO(), []byte(`{"x": 10, "y": 20}`))
	require.NoError(t, err)
	require.Equal(t, []byte(`{"O":30}`), added)

	multiplied, err := multiService.Call(context.TODO(), []byte(`{"x": 10, "y": 20}`))
	require.NoError(t, err)
	require.Equal(t, []byte(`{"O":200}`), multiplied)

}
