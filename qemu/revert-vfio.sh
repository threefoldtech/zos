#!/bin/bash

set -e

echo "[1/5] Reverting GRUB parameters..."

# Clean VFIO-related parameters from GRUB_CMDLINE_LINUX_DEFAULT
sudo sed -i 's/\s*vfio-pci.ids=[^" ]*//g' /etc/default/grub
sudo sed -i 's/\s*intel_iommu=on//g' /etc/default/grub
sudo sed -i 's/\s*iommu=pt//g' /etc/default/grub

echo "[2/5] Removing VFIO initramfs modules..."
sudo rm -f /etc/initramfs-tools/modules

echo "[3/5] Removing NVIDIA blacklist..."
sudo rm -f /etc/modprobe.d/blacklist-nvidia.conf

echo "[4/5] Regenerating initramfs..."
sudo update-initramfs -u

echo "[5/5] Updating GRUB..."
sudo update-grub

echo "✅ Reverted VFIO GPU passthrough configuration. Reboot to re-enable NVIDIA on host."
