package yggdrasil

import (
	"crypto/ed25519"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestAddresses(t *testing.T) {
	sk := ed25519.NewKeyFromSeed([]byte("00000000000000000000000000000000"))

	cfg := GenerateConfig(sk)
	s := NewYggServer(&cfg)

	ip, err := s.Address()
	require.NoError(t, err)

	subnet, err := s.Subnet()
	require.NoError(t, err)

	gw, err := s.Gateway()
	require.NoError(t, err)

	assert.Equal(t, "203:45bf:8a48:8361:c04c:1321:ea32:50ad", ip.String())
	assert.Equal(t, "303:45bf:8a48:8361::/64", subnet.String())
	assert.Equal(t, "303:45bf:8a48:8361::1/64", gw.String())
}

func TestSubnetFor(t *testing.T) {
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
