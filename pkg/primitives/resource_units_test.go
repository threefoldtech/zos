package primitives

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func Test_processZDB(t *testing.T) {
	type args struct {
		r *gridtypes.Workload
	}

	tests := []struct {
		name    string
		args    args
		want    resourceUnits
		wantErr bool
	}{
		{
			name: "zdbSSD",
			args: args{
				r: &gridtypes.Workload{
					Type: zos.ZDBType,
					Data: mustMarshalJSON(t, zos.ZDB{
						Size:     1,
						DiskType: zos.SSDDevice,
					}),
				},
			},
			want: resourceUnits{
				SRU: 1 * gib,
			},
			wantErr: false,
		},
		{
			name: "zdbHDD",
			args: args{
				r: &gridtypes.Workload{
					Type: zos.ZDBType,
					Data: mustMarshalJSON(t, zos.ZDB{
						Size:     1,
						DiskType: zos.HDDDevice,
					}),
				},
			},
			want: resourceUnits{
				HRU: 1 * gib,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processZdb(tt.args.r)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_processVolume(t *testing.T) {
	type args struct {
		r *gridtypes.Workload
	}

	tests := []struct {
		name    string
		args    args
		wantU   resourceUnits
		wantErr bool
	}{
		{
			name: "volumeSSD",
			args: args{
				r: &gridtypes.Workload{
					Type: zos.VolumeType,
					Data: mustMarshalJSON(t, Volume{
						Size: 1,
						Type: zos.SSDDevice,
					}),
				},
			},
			wantU: resourceUnits{
				SRU: 1 * gib,
			},
		},
		{
			name: "volumeHDD",
			args: args{
				r: &gridtypes.Workload{
					Type: zos.VolumeType,
					Data: mustMarshalJSON(t, Volume{
						Size: 1,
						Type: zos.HDDDevice,
					}),
				},
			},
			wantU: resourceUnits{
				HRU: 1 * gib,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotU, err := processVolume(tt.args.r)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.wantU, gotU)
			}
		})
	}
}

func Test_processContainer(t *testing.T) {
	type args struct {
		r *gridtypes.Workload
	}

	tests := []struct {
		name    string
		args    args
		wantU   resourceUnits
		wantErr bool
	}{
		{
			name: "container",
			args: args{
				r: &gridtypes.Workload{
					Type: zos.VolumeType,
					Data: mustMarshalJSON(t, Container{
						Capacity: zos.ContainerCapacity{
							CPU:      2,
							Memory:   1024,
							DiskType: zos.SSDDevice,
							DiskSize: 256,
						},
					}),
				},
			},
			wantU: resourceUnits{
				CRU: 2,
				MRU: 1 * gib,
				SRU: 256 * mib,
			},
		},
		{
			name: "container",
			args: args{
				r: &gridtypes.Workload{
					Type: zos.VolumeType,
					Data: mustMarshalJSON(t, Container{
						Capacity: zos.ContainerCapacity{
							CPU:      2,
							Memory:   2048,
							DiskType: zos.SSDDevice,
							DiskSize: 1024,
						},
					}),
				},
			},
			wantU: resourceUnits{
				CRU: 2,
				MRU: 2 * gib,
				SRU: 1 * gib,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotU, err := processContainer(tt.args.r)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.wantU, gotU)
			}
		})
	}
}

func Test_processKubernetes(t *testing.T) {
	type args struct {
		r *gridtypes.Workload
	}

	tests := []struct {
		name    string
		args    args
		wantU   resourceUnits
		wantErr bool
	}{
		{
			name: "k8sSize1",
			args: args{
				r: &gridtypes.Workload{
					Type: zos.KubernetesType,
					Data: mustMarshalJSON(t, Kubernetes{
						Size: 1,
					}),
				},
			},
			wantU: resourceUnits{
				CRU: 1,
				MRU: 2 * gib,
				SRU: 50 * gib,
			},
		},
		{
			name: "k8sSize2",
			args: args{
				r: &gridtypes.Workload{
					Type: zos.KubernetesType,
					Data: mustMarshalJSON(t, Kubernetes{
						Size: 2,
					}),
				},
			},
			wantU: resourceUnits{
				CRU: 2,
				MRU: 4 * gib,
				SRU: 100 * gib,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotU, err := processKubernetes(tt.args.r)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.wantU, gotU)
			}
		})
	}
}

func mustMarshalJSON(t *testing.T, v interface{}) []byte {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
