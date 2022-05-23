package provision

import (
	"fmt"
	"testing"

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
			in: Paused(),
			out: gridtypes.Result{
				State: gridtypes.StatePaused,
				Error: "",
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%+v", c.in), func(t *testing.T) {
			result, err := buildResult(nil, c.in)
			require.NoError(t, err)

			require.Equal(t, c.out.State, result.State)
			require.Equal(t, c.out.Error, result.Error)
		})
	}
}
