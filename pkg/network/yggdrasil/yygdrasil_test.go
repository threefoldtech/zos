package yggdrasil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/identity"
	"gotest.tools/assert"
)

func TestGenerateConfig(t *testing.T) {
	kp, err := identity.FromSeed([]byte("00000000000000000000000000000000"))
	require.NoError(t, err)

	cfg := GenerateConfig(kp.PrivateKey)

	assert.Equal(t, "30303030303030303030303030303030303030303030303030303030303030301ba4075b77c9e3fb3ecde15cdaf5221f3c10373e623f7b0e1ef76366b0af7137", cfg.SigningPrivateKey)
	assert.Equal(t, "1ba4075b77c9e3fb3ecde15cdaf5221f3c10373e623f7b0e1ef76366b0af7137", cfg.SigningPublicKey)
	assert.Equal(t, "98b6d128682e280b74b324ca82a6bae6e8a3f7174e0605bfd52eb9948fad8944", cfg.EncryptionPrivateKey)
	assert.Equal(t, "167315f5a03796214692d3a74b0a48d630f7fa5e5257730da0415dc7af7ab260", cfg.EncryptionPublicKey)
	assert.Equal(t, "2ru5PcgeQzxF7QZYwQgDkG2K13PRqyigVw99zMYg8eML", cfg.NodeInfo["name"])
}

func TestAddresses(t *testing.T) {
	kp, err := identity.FromSeed([]byte("00000000000000000000000000000000"))
	require.NoError(t, err)

	cfg := GenerateConfig(kp.PrivateKey)
	s := NewServer(nil, &cfg)

	ip, err := s.Address()
	require.NoError(t, err)

	subnet, err := s.Subnet()
	require.NoError(t, err)

	gw, err := s.Gateway()
	require.NoError(t, err)

	assert.Equal(t, "201:8331:6ca1:39c7:f468:fab5:2bb9:daf4", ip.String())
	assert.Equal(t, "301:8331:6ca1:39c7::/64", subnet.String())
	assert.Equal(t, "301:8331:6ca1:39c7::1/64", gw.String())
}
