package provision

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestBuildResult(t *testing.T) {
	type Case struct {
		in  error
		out gridtypes.Result
	}
	cases := []Case{
		{
			in: nil,
			out: gridtypes.Result{
				State: gridtypes.StateOk,
				Error: "",
			},
		},
		{
			in: Ok(),
			out: gridtypes.Result{
				State: gridtypes.StateOk,
				Error: "",
			},
		},
		{
			in: fmt.Errorf("something went wrong"),
			out: gridtypes.Result{
				State: gridtypes.StateError,
				Error: "something went wrong",
			},
		},
		{
			in: UnChanged(fmt.Errorf("failed to update")),
			out: gridtypes.Result{
				State: gridtypes.StateUnChanged,
				Error: "failed to update",
			},
		},
		{
			in: errors.Wrap(UnChanged(fmt.Errorf("failed to update")), "wrapped"),
			out: gridtypes.Result{
				State: gridtypes.StateUnChanged,
				Error: "wrapped: failed to update",
			},
		},
		{
			in: Paused(),
			out: gridtypes.Result{
				State: gridtypes.StatePaused,
				Error: "paused",
			},
		},
		{
			in: errors.Wrap(Paused(), "wrapped for some reason"),
			out: gridtypes.Result{
				State: gridtypes.StatePaused,
				Error: "wrapped for some reason: paused",
			},
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			result, err := buildResult(nil, c.in)
			require.NoError(t, err)

			require.Equal(t, c.out.State, result.State)
			require.Equal(t, c.out.Error, result.Error)
		})
	}
}
