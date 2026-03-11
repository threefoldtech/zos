# Publish Bootstrap

**File:** `.github/workflows/publish-bootstrap.yaml`
**Trigger:** Push to files in `bootstrap/bootstrap/**` or changes to the workflow file itself

## Purpose

Builds and publishes the ZOS bootstrap binary. The bootstrap is a Rust binary compiled with musl for static linking, which is responsible for downloading and starting all ZOS services on node boot.

## Steps

1. **Checkout** the repository
2. **Prepare musl**: Installs `musl` and `musl-tools` for static compilation
3. **Setup Rust toolchain**: Configures stable Rust with `x86_64-unknown-linux-musl` target
4. **Build bootstrap**: Runs `make release` in `bootstrap/bootstrap/`
5. **Collect files**: Copies the built binary to `archive/sbin/bootstrap`
6. **Set name**: Generates a versioned name like `bootstrap-v<YYMMDD.HHMMSS.0>-dev.flist`
7. **Publish flist**: Uploads to `tf-autobuilder/<name>.flist`
8. **Symlink (development)**: Always creates symlink `bootstrap:development.flist` pointing to the new build
9. **Symlink (release)**: On `main` branch only, also creates symlink `bootstrap:latest.flist`
