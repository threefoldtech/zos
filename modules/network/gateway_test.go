package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_gwIP(t *testing.T) {
	type args struct {
		prefix net.IP
		n      int
	}
	tests := []struct {
		name string
		args args
		want net.IP
	}{
		{
			name: "n=1",
			args: args{
				prefix: net.ParseIP("2a02:1802:5e::"),
				n:      1,
			},
			want: net.ParseIP("2a02:1802:5e:1000::1"),
		},
		{
			name: "n=2",
			args: args{
				prefix: net.ParseIP("2a02:1802:5e::"),
				n:      2,
			},
			want: net.ParseIP("2a02:1802:5e:2000::1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gwIP(tt.args.prefix, tt.args.n)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_nrIP(t *testing.T) {
	type args struct {
		prefixZero net.IP
		nrPrefix   net.IP
		n          int
	}
	tests := []struct {
		name string
		args args
		want net.IP
	}{
		{
			name: "n=1",
			args: args{
				prefixZero: net.ParseIP("2a02:1802:5e::"),
				nrPrefix:   net.ParseIP("2a02:1802:5e:d13f::"),
				n:          1,
			},
			want: net.ParseIP("2a02:1802:5e::1000:d13f"),
		},
		{
			name: "n=2",
			args: args{
				prefixZero: net.ParseIP("2a02:1802:5e::"),
				nrPrefix:   net.ParseIP("2a02:1802:5e:d13f::"),
				n:          2,
			},
			want: net.ParseIP("2a02:1802:5e::2000:d13f"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nrIP(tt.args.prefixZero, tt.args.nrPrefix, tt.args.n)
			assert.Equal(t, tt.want, got)
		})
	}
}
