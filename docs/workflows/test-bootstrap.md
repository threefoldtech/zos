# Bootstrap Tests

**File:** `.github/workflows/test-bootstrap.yaml`
**Trigger:** Push to files in `bootstrap/bootstrap/**` or changes to the workflow file itself

## Purpose

Runs tests for the ZOS bootstrap binary (Rust) and verifies it compiles successfully.

## Steps

1. **Checkout** the repository
2. **Prepare musl**: Installs `musl` and `musl-tools`
3. **Setup Rust toolchain**: Configures stable Rust with `x86_64-unknown-linux-musl` target
4. **Test**: Runs `make test` in `bootstrap/bootstrap/`
5. **Build**: Runs `make release` to verify the release build compiles
