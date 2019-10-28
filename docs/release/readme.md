# Releases of Zero-OS
We use a simple but powerful pipeline release workflow. Building and file distribution are made using GitHub Actions.
Usable files are available on the [Zero-OS Hub](https://playground.hub.grid.tf/tf-zos).

## Development build
On any push on the zos repository, a new development build is triggered. If the build succeed,
binaries are packed into an flist and uploaded to the `tf-autobuilder` repository of the hub.

A symlink to this latest build is made into the same repository (`tf-autobuilder/zos:development:latest.flist`)

## Testing build
As soon as a version seems good to be tested using our testnet, we will produce a release within GitHub.
This release will be tagged `vX.Y.Z-something` (eg: `v2.0.4-beta3`). Like development builds, everything is compiled
and uploaded, but in addition, this file will be copied to `tf-zos/zos-version.flist` and will be symlinked then
to `tf-zos/zos:testing:latest.flist`.

## Production build
As soon as a build is bullet-proof tested and working fine, a new release will be made within GitHub and this
release will be tagged `vX.Y.Z`. This is the final versioning form.

Like for testing, everything is uploaded and symlinked, but now using `zos:production:latest.flist` filename.

# Always Up-to-date
If you want to always uses the latest up-to-date build of our releases, you should uses theses files:

- Development: https://playground.hub.grid.tf/tf-autobuilder/zos:development:latest.flist
- Testing: https://playground.hub.grid.tf/tf-zos/zos:testing:latest.flist
- Production: https://playground.hub.grid.tf/tf-zos/zos:production:latest.flist
