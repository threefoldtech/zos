package nr

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"github.com/vishvananda/netlink"
)

type testIdentityManager struct {
	id   string
	farm uint64
}

var _ pkg.IdentityManager = (*testIdentityManager)(nil)

func (t *testIdentityManager) StoreKind() string {
	return "test"
}

// NodeID returns the node id (public key)
func (t *testIdentityManager) NodeID() pkg.StrIdentifier {
	return pkg.StrIdentifier(t.id)
}

func (t *testIdentityManager) Address() (pkg.Address, error) {
	return pkg.Address(t.id), nil
}

func (t *testIdentityManager) Farm() (string, error) {
	return "test-farm", nil
}

func (t *testIdentityManager) FarmSecret() (string, error) {
	return "", nil
}

// FarmID return the farm id this node is part of. this is usually a configuration
// that the node is booted with. An error is returned if the farmer id is not configured
func (t *testIdentityManager) FarmID() (pkg.FarmID, error) {
	return pkg.FarmID(t.farm), nil
}

// Sign signs the message with privateKey and returns a signature.
func (t *testIdentityManager) Sign(message []byte) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Verify reports whether sig is a valid signature of message by publicKey.
func (t *testIdentityManager) Verify(message, sig []byte) error {
	return fmt.Errorf("not implemented")
}

// Encrypt encrypts message with the public key of the node
func (t *testIdentityManager) Encrypt(message []byte) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Decrypt decrypts message with the private of the node
func (t *testIdentityManager) Decrypt(message []byte) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// EncryptECDH aes encrypt msg using a shared key derived from private key of the node and public key of the other party using Elliptic curve Diffie Helman algorithm
// the nonce if prepended to the encrypted message
func (t *testIdentityManager) EncryptECDH(msg []byte, publicKey []byte) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// DecryptECDH decrypt aes encrypted msg using a shared key derived from private key of the node and public key of the other party using Elliptic curve Diffie Helman algorithm
func (t *testIdentityManager) DecryptECDH(msg []byte, publicKey []byte) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// PrivateKey sends the keypair
func (t *testIdentityManager) PrivateKey() []byte {
	return nil
}

func TestNamespace(t *testing.T) {
	nr := New(pkg.Network{}, "")

	nsName, err := nr.Namespace()
	require.NoError(t, err)

	brName, err := nr.BridgeName()
	require.NoError(t, err)

	wgName, err := nr.WGName()
	require.NoError(t, err)

	assert.Equal(t, "net-networkd1", nsName)
	assert.Equal(t, "br-networkd1", brName)
	assert.Equal(t, "wg-networkd1", wgName)
}

func TestNaming(t *testing.T) {
	nr := New(pkg.Network{}, "")

	nsName, err := nr.Namespace()
	require.NoError(t, err)

	brName, err := nr.BridgeName()
	require.NoError(t, err)

	wgName, err := nr.WGName()
	require.NoError(t, err)

	assert.Equal(t, "net-networkd1", nsName)
	assert.Equal(t, "br-networkd1", brName)
	assert.Equal(t, "wg-networkd1", wgName)
}

func TestCreateBridge(t *testing.T) {
	nr := New(pkg.Network{}, "")

	brName, err := nr.BridgeName()
	require.NoError(t, err)

	err = nr.ensureNRBridge()
	require.NoError(t, err)

	l, err := netlink.LinkByName(brName)
	assert.NoError(t, err)
	_, ok := l.(*netlink.Bridge)
	assert.True(t, ok)

	// cleanup
	_ = netlink.LinkDel(l)
}

func Test_wgIP(t *testing.T) {
	type args struct {
		subnet *net.IPNet
	}
	tests := []struct {
		name string
		args args
		want *net.IPNet
	}{
		{
			name: "default",
			args: args{
				subnet: &net.IPNet{
					IP:   net.ParseIP("10.3.1.0"),
					Mask: net.CIDRMask(16, 32),
				},
			},
			want: &net.IPNet{
				IP:   net.ParseIP("100.64.3.1"),
				Mask: net.CIDRMask(16, 32),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wgIP(tt.args.subnet); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("wgIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convert4to6(t *testing.T) {
	type args struct {
		netID string
		ip    net.IP
	}
	tests := []struct {
		name string
		args args
		want net.IP
	}{
		{
			name: "valid",
			args: args{
				netID: "networkdID",
				ip:    net.ParseIP("100.127.0.2"),
			},
			want: net.ParseIP("fd6e:6574:776f:0000::2"),
		},
		{
			name: "valid",
			args: args{
				netID: "networkdID",
				ip:    net.ParseIP("100.127.2.16"),
			},
			want: net.ParseIP("fd6e:6574:776f:0002::0010"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.want)
			got := Convert4to6(tt.args.netID, tt.args.ip)
			require.NotNil(t, got)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestMyceliumGw(t *testing.T) {
	data := MyceliumInspection{
		Address: net.ParseIP("3b4:ca67:822d:b0c1:5d6f:c647:1ed8:6ced"),
	}

	subnet, gw, err := data.Gateway()
	require.NoError(t, err)

	require.Equal(t, net.ParseIP("3b4:ca67:822d:b0c1::1"), gw.IP)
	require.Equal(t, "3b4:ca67:822d:b0c1::1/64", gw.String())
	require.Equal(t, "3b4:ca67:822d:b0c1::/64", subnet.String())

}
func TestMyceliumIP(t *testing.T) {
	data := MyceliumInspection{
		Address: net.ParseIP("3b4:ca67:822d:b0c1:5d6f:c647:1ed8:6ced"),
	}

	ip, gw, err := data.IP([]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
	require.NoError(t, err)

	require.Equal(t, net.ParseIP("3b4:ca67:822d:b0c1:ffff:11:2233:4455"), ip.IP)
	require.Equal(t, "3b4:ca67:822d:b0c1::1/64", gw.String())

}
