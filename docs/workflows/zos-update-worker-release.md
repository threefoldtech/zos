# ZOS Update Worker (Release)

**File:** `.github/workflows/zos-update-worker-release.yml`
**Trigger:** Push of version tags (`v*`)

## Purpose

Builds and publishes the ZOS update worker binary as a GitHub release asset when a new version tag is pushed.

## Steps

1. **Checkout** the repository
2. **Install Go 1.19**
3. **Build**: Runs `make build` in `tools/zos-update-worker/`
4. **Get release**: Fetches the GitHub release matching the pushed tag
5. **Upload release asset**: Attaches `bin/zos-update-worker` to the GitHub release
