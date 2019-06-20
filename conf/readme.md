
## Services Boot Sequence
- node-read
  - storaged
  - netowkrd
- boot
  - contd
  - flister
  - upgrade

## Pseudo boot steps
both node-read and boot are not actual services, but instead they are there to define a `boot stage`. for example once `node-ready` service is (ready) it means all crucial system services defined by 0-initramfs is now running. This now only containes the `udev` daemons, but it's easy to add more without the need to change all the other services.

`boot` srevice is similar, but grantees that some zos2 services are running (for example `storaged`) before start other services like flister which requires storaged
