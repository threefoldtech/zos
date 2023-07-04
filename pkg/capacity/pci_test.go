package capacity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDevice(t *testing.T) {
	t.Run("get GPU without SSID", func(t *testing.T) {
		vendor, device, ok := GetDevice(0x1002, 0x731f, 0, 0)

		require.True(t, ok)
		require.Equal(t, "Advanced Micro Devices, Inc. [AMD/ATI]", vendor.Name)
		require.Equal(t, "Navi 10 [Radeon RX 5600 OEM/5600 XT / 5700/5700 XT]", device.Name)

	})

	t.Run("get GPU with SSID", func(t *testing.T) {
		vendor, device, ok := GetDevice(0x10de, 0x1e30, 0x10de, 0x129e)

		require.True(t, ok)
		require.Equal(t, "NVIDIA Corporation", vendor.Name)
		require.Equal(t, "Quadro RTX 8000", device.Name)
	})
}

func TestListPCI(t *testing.T) {
	devices, err := ListPCI()
	require.NoError(t, err)

	for _, device := range devices {
		fmt.Println(device)
	}
}

func TestListGPU(t *testing.T) {
	devices, err := ListPCI(GPU)
	require.NoError(t, err)

	for _, device := range devices {
		fmt.Println(device)
	}
}
