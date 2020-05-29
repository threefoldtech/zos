package yggdrasil

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/threefoldtech/zos/pkg/identity"
	"gotest.tools/assert"
)

func TestGenerateConfig(t *testing.T) {
	kp, err := identity.FromSeed([]byte("00000000000000000000000000000000"))
	require.NoError(t, err)

	s := NewServer(nil, kp.PrivateKey)

	buf := &bytes.Buffer{}

	err = s.generateConfig(buf)
	require.NoError(t, err)

	config := config{}
	err = json.NewDecoder(buf).Decode(&config)
	require.NoError(t, err)
	assert.Equal(t, "30303030303030303030303030303030303030303030303030303030303030301ba4075b77c9e3fb3ecde15cdaf5221f3c10373e623f7b0e1ef76366b0af7137", config.SigningPrivateKey)
	assert.Equal(t, "1ba4075b77c9e3fb3ecde15cdaf5221f3c10373e623f7b0e1ef76366b0af7137", config.SigningPublicKey)
	assert.Equal(t, "98b6d128682e280b74b324ca82a6bae6e8a3f7174e0605bfd52eb9948fad8944", config.EncryptionPrivateKey)
	assert.Equal(t, "167315f5a03796214692d3a74b0a48d630f7fa5e5257730da0415dc7af7ab260", config.EncryptionPublicKey)
	assert.Equal(t, "2ru5PcgeQzxF7QZYwQgDkG2K13PRqyigVw99zMYg8eML", config.NodeInfo.Name)
}
