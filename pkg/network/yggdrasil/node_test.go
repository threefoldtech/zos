package yggdrasil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestNode_SubnetFor(t *testing.T) {
	type fields struct {
		prefix net.IP
		ip     net.IP
		b      []byte
	}
	tests := []struct {
		name   string
		prefix net.IP
		ip     net.IP
		b      []byte
	}{
		{
			name:   "0000",
			prefix: net.ParseIP("301:8d90:378f:a93::"),
			ip:     net.ParseIP("301:8d90:378f:a93:c600:1d5b:2ac3:df31"),
			b:      []byte("0000"),
		},
		{
			name:   "bbbbb",
			prefix: net.ParseIP("301:8d90:378f:a93::"),
			ip:     net.ParseIP("301:8d90:378f:a93:9b73:d9aa:7cec:73be"),
			b:      []byte("bbbb"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := subnetFor(tt.prefix, tt.b)
			require.NoError(t, err)
			assert.Equal(t, tt.ip.String(), ip.String())
		})
	}
}
