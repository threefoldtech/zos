# Generic Virtual Machines

A generic VM consists of three files:
- image.raw: raw disk image
- kernel: The kernel executable file
- initrd (optional)

## Image file

The raw disk image contains the filesystem in the btrfs format of the VM.

Example of constructing a raw image:
```
wget http://cdimage.ubuntu.com/ubuntu-base/releases/20.04.1/release/ubuntu-base-20.04.1-base-amd64.tar.gz
truncate -s 2G image.raw    # Create a blank raw disk
mkfs -t btrfs image.raw     # Create a btrfs filesystem on it
mkdir mnt
mount -o loop image.raw mnt # mount it on the ./mnt directory
cd mnt && tar zxvf ../ubuntu-base-20.04.1-base-amd64.tar.gz # Extract the ubuntu base image in the fs
btrfs filesystem resize 1G mnt # Resize the filesystem to avoid non necessary large image.raw
umount mnt                     # Unmount it
truncate -s 1G image.raw       # Resize the disk
```

The image must have the `ip` utility installed to configure the network. It should also contain an ssh server enabled after boot to allow user access.

## Initrd

When provided. The initrd receives in its cmdline a list string of whitespace separated key-value entries. It can be used to create the authorized keys and configure the network interfaces. The ssh entries is in the format "ssh=key". The whitespaces in the ssh key is replaced with ",", so something like `$(echo "$SSH" | sed 's/,/ /g')` should be done before appending it to the authorized keys. The network parameters are processed using the [setupnetwork](https://raw.githubusercontent.com/threefoldtech/k3os/zos-patch/overlay/sbin/setupnetwork) script.

## Example

[Ubuntu focal](https://hub.grid.tf/omar0.3bot/omarelawady-zos-ubuntu-vm-latest.flist.md)