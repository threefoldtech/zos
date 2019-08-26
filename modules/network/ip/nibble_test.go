package ip

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNibble(t *testing.T) {
	type args struct {
		prefix  *net.IPNet
		allocNr int8
	}
	tests := []struct {
		name    string
		args    args
		want    *Nibble
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				prefix:  mustParseCIDR("2a02:1802:5e:ff02::/48"),
				allocNr: 1,
			},
			want: &Nibble{
				nibble:  []byte{0xff, 0x02},
				allocNr: 1,
			},
			wantErr: false,
		},
		{
			name: "wrong-size-smaller",
			args: args{
				prefix:  mustParseCIDR("2a02:1802:5e:ff02::/44"),
				allocNr: 1,
			},
			wantErr: true,
		},
		{
			name: "wrong-size-bigger",
			args: args{
				prefix:  mustParseCIDR("2a02:1802:5e:ff02::/44"),
				allocNr: 1,
			},
			wantErr: true,
		},
		{
			name: "prefix nil",
			args: args{
				prefix:  nil,
				allocNr: 1,
			},
			wantErr: true,
		},
		{
			name: "alloc negative",
			args: args{
				prefix:  mustParseCIDR("2a02:1802:5e:ff02::/44"),
				allocNr: -1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNibble(tt.args.prefix, tt.args.allocNr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestHex(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.Hex()
	assert.Equal(t, "ff02", actual)
}
func TestWiregardName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WiregardName()
	assert.Equal(t, "wg-ff02-0", actual)
}
func TestWireguardPort(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WireguardPort()
	assert.Equal(t, uint16(65282), actual)
}
func TestBridgeName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.BridgeName()
	assert.Equal(t, "br-ff02-0", actual)
}
func TestNetworkName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.NetworkName()
	assert.Equal(t, "net-ff02-0", actual)
}
func TestVethName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.VethName()
	assert.Equal(t, "veth-ff02-0", actual)
}
func TestPubName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.PubName()
	assert.Equal(t, "pub-ff02-0", actual)
}
func TestExitFe80(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.ExitFe80()
	assert.Equal(t, "fe80::ff02/64", actual.String())
}
func TestExitPrefixZero(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	prefixZero := &net.IPNet{
		IP:   net.ParseIP("2a02:1802:5e::"),
		Mask: net.CIDRMask(48, 128),
	}
	actual := nibble.ExitPrefixZero(prefixZero)
	assert.Equal(t, "2a02:1802:5e::ff02/64", actual.String())
}
func TestNRIPv4(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.NRIPv4()
	assert.Equal(t, "10.255.2.1/24", actual.String())
}
func TestWGIP(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WGIP()
	assert.Equal(t, "10.255.255.2/16", actual.String())
}
func TestWGAllowedIP(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WGAllowedIP()
	assert.Equal(t, "10.255.255.2/24", actual.String())
}
func TestWGAllowedFE80(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WGAllowedFE80()
	assert.Equal(t, "fe80::ff02/128", actual.String())
}
func TestWGRouteGateway(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.WGRouteGateway()
	assert.Equal(t, "fe80::ff02", actual.String())
}
func TestRouteIPv6Exit(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.RouteIPv6Exit()
	assert.Equal(t, "::/0", actual.Dst.String())
	assert.Equal(t, "fe80::ff02", actual.Gw.String())
}
func TestRouteIPv4Exit(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.RouteIPv4Exit()
	assert.Equal(t, "10.255.2.0/24", actual.Dst.String())
	assert.Equal(t, "10.255.255.2", actual.Gw.String())
}
func TestRouteIPv4DefaultExit(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.RouteIPv4DefaultExit()
	assert.Equal(t, "0.0.0.0/0", actual.Dst.String())
	assert.Equal(t, "10.255.255.2", actual.Gw.String())
}
func TestToGWName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.ToGWName()
	assert.Equal(t, "to-ff02-0", actual)
}
func TestGWtoNRName(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.GWtoNRName()
	assert.Equal(t, "nr-ff02-0", actual)
}
func TestGWLinkLocal(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.GWLinkLocal(1)
	assert.Equal(t, "fe80::", actual.String())
}

func TestGWIP(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.GWIP(prefix.IP, 1)
	assert.Equal(t, "", actual)
}
func TestGWNRLinkLocal(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.GWNRLinkLocal()
	assert.Equal(t, "", actual)
}
func TestGWNRIP(t *testing.T) {
	prefix := mustParseCIDR("2a02:1802:5e:ff02::/48")
	prefixZero := &net.IPNet{
		IP:   net.ParseIP("2a02:1802:5e::"),
		Mask: net.CIDRMask(48, 128),
	}
	nibble, _ := NewNibble(prefix, 0)
	actual := nibble.GWNRIP(prefixZero.IP, 1)
	assert.Equal(t, "", actual)
}

func mustParseCIDR(cidr string) *net.IPNet {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	ipnet.IP = ip
	return ipnet
}
