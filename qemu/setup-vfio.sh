#!/bin/bash

set -e

echo "[1/5] Updating GRUB boot parameters..."

# Backup and update GRUB
# Update vfio-pci.ids with your GPU IDs 
sudo sed -i.bak '/^GRUB_CMDLINE_LINUX_DEFAULT=/ s/"$/ intel_iommu=on iommu=pt vfio-pci.ids=10de:2560,10de:228e"/' /etc/default/grub

echo "[2/5] Creating vfio.conf for initramfs..."
echo -e "vfio\nvfio_iommu_type1\nvfio_pci\nvfio_virqfd" | sudo tee /etc/initramfs-tools/modules

echo "[3/5] Blacklisting NVIDIA drivers..."
cat <<EOF | sudo tee /etc/modprobe.d/blacklist-nvidia.conf
blacklist nouveau
blacklist nvidia
blacklist nvidia_drm
blacklist nvidia_modeset
blacklist nvidia_uvm
EOF

echo "[4/5] Regenerating initramfs..."
sudo update-initramfs -u

echo "[5/5] Updating GRUB..."
sudo update-grub

echo "✅ All done. Please reboot your system for changes to take effect."
