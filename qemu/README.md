# Run 0-OS in a VM using qemu

This folder contains a script that you can use to run 0-OS in a VM using qemu.

For ease of use a Makefile is also provided. To prepare your environment call `make prepare`. This will :

- copy `zinit` binary into the overlay
- download a 0-OS kernel

To start the 0-OS VM, do `make start`