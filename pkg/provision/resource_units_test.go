package provision

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
)

func Test_processZDB(t *testing.T) {
	type args struct {
		r *Reservation
	}

	zdb := ZDB{
		Size:     1,
		DiskType: pkg.SSDDevice,
	}
	zdbEncoded, err := json.Marshal(zdb)
	require.NoError(t, err)
	t.Logf("%s", string(zdbEncoded))

	tests := []struct {
		name    string
		args    args
		wantU   resourceUnits
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				r: &Reservation{
					Type: ZDBReservation,
					Data: zdbEncoded,
				},
			},
			wantU: resourceUnits{
				SRU: 1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotU, err := processZdb(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("processZdb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotU, tt.wantU) {
				t.Errorf("processZdb() = %v, want %v", gotU, tt.wantU)
			}
		})
	}
}
