# Zmachine

A `Zmachine` is an instance of virtual compute capacity. There are 2 kinds of Zmachines.
One is a `VM`, standard in cloud environments. Next to this it can also be a `container`.
On the Zos level, both of these are implemented as virtual machines. Depending on
the context, it will be considered to be either a VM or a container. In either
scenario, the `Zmachine` is started from an `Flist`.

> Note, both VM and Container on ZOS are actually served as Virtual Machines. The
only difference is that if you are running in VM mode, you only need to provide a **raw**
disk image (image.raw) in your flist.

## Container

A container is meant to host `microservice`. The `microservice` architecture generally
dictates that each service should be run in it's own container (therefore providing
a level of isolation), and communicate with other containers it depends on over the
network.

Similar to docker. In Zos, a container is actually also run in a virtualized environment.
Similar to containers, some setup is done on behalf of the user. After setup this is done,
the users `entrypoint` is started.

It should be noted that a container has no control over the kernel
used to run it, if this is required, a `VM` should be used instead. Furthermore,
a container should ideally only have 1 process running. A container can be a single
binary, or a complete filesystem. In general, the first should be preferred, and
if you need the latter, it might be an indication that you actually want a `VM`.

For containers, the network setup will be created for you. Your init process can
assume that it will be fully set up (according to the config you provided) by the
time it is started. Mountpoints will also be setup for you. The environment variables
passed will be available inside the container.

## VM

In container mode, zos provide a minimal kernel that is used to run a light weight VM
and then run your app from your flist. If you need control over the kernel you can actually
still provide it inside the flist as follows:

- /boot/vmlinuz
- /boot/initrd.img [optional]

**NOTE**: the vmlinuz MUST be an EFI kernel (not compressed) if building your own kernel, or you can use the [extract-vmlinux](https://github.com/torvalds/linux/blob/master/scripts/extract-vmlinux) script to extract the EFI kernel. To test if your kernel is a valid elf kernel run command
`readelf -n <path/to/vmlinuz>`

Any of those files can be a symlink to another file in the flist.

If ZOS found the `/boot/vmlinuz` file, it will use this with the initrd.img if also exists. otherwise zos will use the built-in minimal kernel and run in `container` mode.

### Building an ubuntu VM flist

This is a guide to help you build a working VM flist.

This guide is for ubuntu `jammy`

prepare rootfs

```bash
mkdir ubuntu:jammy
```

bootstrap ubuntu

```bash
sudo debootstrap jammy ubuntu:jammy  http://archive.ubuntu.com/ubuntu
```

this will create and download the basic rootfs for ubuntu jammy in the directory `ubuntu:jammy`.
After its done we can then chroot into this directory to continue installing the necessary packages needed and configure
few things.

> I am using script called `arch-chroot` which is available by default on arch but you can also install on ubuntu to continue
the following steps

```bash
sudo arch-chroot ubuntu:jammy
```

> This script (similar to the `chroot` command) switch root to that given directory but also takes care of mounting /dev /sys, etc.. for you
and clean it up on exit.

Next just remove this link and re-create the file with a valid name to be able to continue

```bash
# make sure to set the path correctly
export PATH=/usr/local/sbin/:/usr/local/bin/:/usr/sbin/:/usr/bin/:/sbin:/bin

rm /etc/resolv.conf
echo 'nameserver 1.1.1.1' > /etc/resolv.conf
```

Install cloud-init

```bash
apt-get update
apt-get install cloud-init openssh-server curl
# to make sure we have clean setup
cloud-init clean
```

Also really important that we install a kernel

```bash
apt-get install linux-modules-extra-5.15.0-25-generic
```

> I choose this package because it will also install extra modules for us and a generic kernel

Next make sure that virtiofs is part of the initramfs image

```bash
echo 'fs-virtiofs' >> /etc/initramfs-tools/modules
update-initramfs -c -k all
```

clean up cache

```bash
apt-get clean
```

Last thing we do inside the container before we actually upload the flist
is to make sure the kernel is in the correct format

This step does not require that we stay in the chroot so hit `ctr+d` or type `exit`

you should be out of the arch-chroot now

```bash
curl -O https://raw.githubusercontent.com/torvalds/linux/master/scripts/extract-vmlinux
chmod +x extract-vmlinux

sudo ./extract-vmlinux ubuntu:jammy/boot/vmlinuz | sudo tee ubuntu:jammy/boot/vmlinuz-5.15.0-25-generic.elf > /dev/null
# then replace original kernel
sudo mv ubuntu:jammy/boot/vmlinuz-5.15.0-25-generic.elf ubuntu:jammy/boot/vmlinuz-5.15.0-25-generic
```

To verify you can do this:

```bash
ls -l ubuntu:jammy/boot
```

and it should show something like

```bash
total 101476
-rw-r--r-- 1 root root   260489 Mar 30  2022 config-5.15.0-25-generic
drwxr-xr-x 1 root root       54 Jun 28 15:35 grub
lrwxrwxrwx 1 root root       28 Jun 28 15:35 initrd.img -> initrd.img-5.15.0-25-generic
-rw-r--r-- 1 root root 41392462 Jun 28 15:39 initrd.img-5.15.0-25-generic
lrwxrwxrwx 1 root root       28 Jun 28 15:35 initrd.img.old -> initrd.img-5.15.0-25-generic
-rw------- 1 root root  6246119 Mar 30  2022 System.map-5.15.0-25-generic
lrwxrwxrwx 1 root root       25 Jun 28 15:35 vmlinuz -> vmlinuz-5.15.0-25-generic
-rw-r--r-- 1 root root 55988436 Jun 28 15:50 vmlinuz-5.15.0-25-generic
lrwxrwxrwx 1 root root       25 Jun 28 15:35 vmlinuz.old -> vmlinuz-5.15.0-25-generic
```

Now package the tar for upload

```bash
sudo rm -rf ubuntu:jammy/dev/*
sudo tar -czf ubuntu-jammy.tar.gz -C ubuntu:jammy .
```

Upload to the hub, and use it to create a Zmachine

## VM Image [deprecated]

In a VM image mode, you run your own operating system (for now only linux is supported)
The image provided must be

- EFI bootable
- Cloud-init enabled.

You can find later in this document how to create your own bootable image.

A VM reservations must also have at least 1 volume, as the boot image
will be copied to this volume. The size of the root disk will be the size of this
volume.

The image used to the boot the VM must has cloud-init enabled on boot. Cloud-init
receive its config over the NoCloud source. This takes care of setting up networking, hostname
, root authorized_keys.

> This method of building a full VM from a raw image is not recommended and will get phased out in
the future. It's better to use either the container method to run containerized Apps. Another option
is to run your own kernel from an flist (explained below)

### Expected Flist structure

An `Zmachine` will be considered a `VM` if it contains an `/image.raw` file.

`/image.raw` is used as "boot disk". This `/image.raw` is copied to the first attached
volume of the `VM`. Cloud-init will take care of resizing the filesystem on the image
to take the full disk size allocated in the deployment.

Note if the `image.raw` size is larger than the allocated disk. the workload for the VM
will fail.

### Expected Flist structure

Any Flist will boot as a container, **UNLESS** is has a `/image.raw` file. There is
no need to specify a kernel yourself (it will be provided).

### Known issues

- We need to do proper performance testing for `virtio-fs`. There seems to be some
    suboptimal performance right now.
- It's not currently possible to get container logs.
- TODO: more testing

## Creating VM image

This is a simple tutorial on how to create your own VM image
> Note: Please consider checking the official vm images repo on the hub before building your own
image. this can save you a lot of time (and network traffic) here <https://hub.grid.tf/tf-official-vms>

### Use one of ubuntu cloud-images

If the ubuntu images in the official repo are not enough, you can simply upload one of the official images as follows

- Visit <https://cloud-images.ubuntu.com/>
- Select the version you want (let's assume bionic)
- Go to bionic, then click on current
- download the amd64.img file like this one <https://cloud-images.ubuntu.com/bionic/current/bionic-server-cloudimg-amd64.img>
- This is a `Qcow2` image, this is not supported by zos. So we need to convert this to a raw disk image using the following command

```bash
qemu-img convert -p -f qcow2 -O raw bionic-server-cloudimg-amd64.img image.raw
```

- now we have the raw image (image.raw) time to compress and upload to the hub

```bash
tar -czf ubuntu-18.04-lts.tar.gz image.raw
```

- now visit the hub <https://hub.grid.tf/> and login or create your own account, then click on upload my file button
- Select the newly created tar.gz file
- Now you should be able to use this flist to create Zmachine workloads

### Create an image from scratch

This is an advanced scenario and you will require some prior knowledge of how to create local VMs and how to prepare the installation medium,
and installing your OS of choice.

Before we continue you need to have some hypervisor that you can use locally. Libvirt/Qemu are good choices. Hence we skip on what you need to do to install and configure your system correctly not how to create the VM

#### VM Requirements

Create a VM with enough CPU and Memory to handle the installation process note that this does not relate on what your choices for CPU and Memory are going to be for the actual VM running on the grid.

We going to install arch linux image. So we will have to create a VM with

- Disk of about 2GB (note this also is not related to the final VM running on the grid, on installation the OS image will expand to use the entire allocated disk attached to the VM eventually). The smaller the disk is better this can be different for each OS.
- Add the arch installation iso or any other installation medium

#### Boot the VM (locally)

Boot the VM to start installation. The boot must support EFI booting because ZOS only support images with esp partition. So make sure that both your hypervisor and boot/installation medium supports this.

For example in Libvirt Manager make sure you are using the right firmware (UEFI)

#### Installation

We going to follow the installation manual for Arch linux but with slight tweaks:

- Make sure VM is booted with UEFI, run `efivar -l` command see if you get any output. Otherwise the machine is probably booted in BIOS mode.
- With `parted` create 2 partitions
  - an esp (boot) partition of 100M
  - a root partition that spans the remaining of the disk

```bash
DISK=/dev/vda
# First, create a gpt partition table
parted $DISK mklabel gpt
# Secondly, create the esp partition of 100M
parted $DISK mkpart primary 1 100M
# Mark first part as esp
parted $DISK set 1 esp on
# Use the remaining part as root that takes the remaining
# space on disk
parted $DISK mkpart primary 100M 100%

# To verify everything is correct do
parted $DISK print

# this should 2 partitions the first one is slightly less that 100M and has flags (boot, esp), the second one takes the remaining space
```

We need to format the partitions as follows:

```bash
# this one has to be vfat of size 32 as follows
mkfs.vfat -F 32 /dev/vda1
# This one can be anything based on your preference as long as it's supported by you OS kernel. we going with ext4 in this tutorial
mkfs.ext4 -L cloud-root /dev/vda2
```

Note the label assigned to the /dev/vda2 (root) partition this can be anything but it's needed to configure the boot later when installing the boot loader. Otherwise you can use the partition UUID.

Next, we need to mount the disks

```bash
mount /dev/vda2 /mnt
mkdir /mnt/boot
mount /dev/vda1 /mnt/boot
```

After disks are mounted as above, we need to start the installation

```bash
pacstrap /mnt base linux linux-firmware vim openssh cloud-init cloud-guest-utils
```

This will install basic arch linux but will also include cloud-init, cloud-guest-utils, openssh, and vim for convenience.

Following the installation guid to generate fstab file

```
genfstab -U /mnt >> /mnt/etc/fstab
```

And arch-chroot into /mnt `arch-chroot /mnt` to continue the setup. please follow all steps in the installation guide to set timezone, and locales as needed.

- You don't have to set the hostname, this will be setup later on zos when the VM is deployed via cloud-init
- let's drop the root password all together since login to the VM over ssh will require key authentication only, you can do this by running

```bash
passwd -d root
```

We make sure required services are enabled

```bash
systemctl enable sshd
systemctl enable systemd-networkd
systemctl enable systemd-resolved
systemctl enable cloud-init
systemctl enable cloud-final

# make sure we using resolved
rm /etc/resolv.conf
ln -s /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
```

Finally installing the boot loader as follows
> Only grub2 has been tested and known to work.

```bash
pacman -S grub
```

Then we need to install grub

```
grub-install --target=x86_64-efi --efi-directory=esp --removable
```

Change default values as follows

```
vim /etc/default/grub
```

And make sure to change `GRUB_CMDLINE_LINUX_DEFAULT` as follows

```
GRUB_CMDLINE_LINUX_DEFAULT="loglevel=3 console=tty console=ttyS0"
```

> Note: we removed the `quiet` and add the console flags.

Also set the `GRUB_TIMEOUT` to 0 for a faster boot

```
GRUB_TIMEOUT=0
```

Then finally generating the config

```
grub-mkconfig -o /boot/grub/grub.cfg
```

Last thing we need to do is clean up

- pacman cache by running `rm -rf /var/cache/pacman/pkg`
- cloud-init state by running `cloud-init clean`

Click `Ctrl+D` to exit the change root, then power off by running `poweroff` command.

> NOTE: if you booted the machine again you always need to do `cloud-init clean` as long as it's not yet deployed on ZOS this to make sure the image has a clean state
>
#### Converting the disk

Based on your hypervisor of choice you might need to convert the disk to a `raw` image same way we did with ubuntu image.

```bash
# this is an optional step in case you used a qcoq disk for the installation. If the disk is already `raw` you can skip this
qemu-img convert -p -f qcow2 -O raw /path/to/vm/disk.img image.raw
```

Compress and tar the image.raw as before, and upload to the hub.

```
tar -czf arch-linux.tar.gz image.raw
```
