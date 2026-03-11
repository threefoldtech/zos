# Tests and Coverage

**File:** `.github/workflows/test.yaml`
**Trigger:** Push to any branch

## Purpose

Runs tests and verifies the ZOS Go codebase compiles. Currently the test steps are commented out — only a build verification is performed.

## Steps

1. **Set up Go 1.23**
2. **Prepare dependencies**: Installs `libjansson-dev` and `libhiredis-dev`
3. **Checkout** the repository
4. **Build binaries**: Runs `make` in `cmds/` to verify compilation

## Note

The actual test execution (`make testrace`) and dependency fetching (`make getdeps`) steps are currently commented out in the workflow.
