package wireguard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// func TestConfigCmd(t *testing.T) {
// 	cmd := configCmd("wg0", "/etc/wg/key.priv", []Peer{
// 		{
// 			PublicKey:  "KF6yeDYqnVquTHiLjUvNDylqCzXpBNSBnFCnC0TBm1M=",
// 			Endpoint:   "37.187.124.71:51820",
// 			AllowedIPs: []string{"0.0.0.0/0"},
// 		},
// 	})

// 	expected := "set wg0 private-key /etc/wg/key.priv peer KF6yeDYqnVquTHiLjUvNDylqCzXpBNSBnFCnC0TBm1M= endpoint 37.187.124.71:51820 allowed-ips 0.0.0.0/0"
// 	assert.Equal(t, expected, cmd)
// }
func TestNewPeer(t *testing.T) {
	_, err := newPeer("mR5fBXohKe2MZ6v+GLwlKwrvkFxo1VvV3bPNHDBhOAI=", "37.187.124.71:51820", []string{"172.21.0.0/24"})
	require.NoError(t, err)

}
