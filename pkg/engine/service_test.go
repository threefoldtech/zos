package engine

import (
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

	add := func(ctx Context, input Input) (Output, error) {
		return Output{O: input.X + input.Y}, nil
	}

	multi := func(ctx Context, input Input) (Output, error) {
		return Output{O: input.X * input.Y}, nil
	}

	addService := NewAction(add).Into()
	multiService := NewAction(multi).Into()

	added, err := addService.Call(&engineContext{}, []byte(`{"x": 10, "y": 20}`))
	require.NoError(t, err)
	require.Equal(t, []byte(`{"O":30}`), added)

	multiplied, err := multiService.Call(&engineContext{}, []byte(`{"x": 10, "y": 20}`))
	require.NoError(t, err)
	require.Equal(t, []byte(`{"O":200}`), multiplied)

}
