# ZOS Update Worker (CI)

**File:** `.github/workflows/zos-update-worker-main.yml`
**Trigger:**
- Push to files in `tools/zos-update-worker/**`
- Pull requests touching `tools/zos-update-worker/**`

## Purpose

Runs linting, static analysis, formatting checks, and tests for the ZOS update worker tool.

## Steps

1. **Checkout** the repository
2. **Install Go 1.19**
3. **golangci-lint**: Runs the golangci linter with a 3-minute timeout
4. **staticcheck**: Runs static analysis (version 2022.1.3)
5. **gofmt**: Checks for formatting issues
6. **Test**: Runs `go test -v ./...`
