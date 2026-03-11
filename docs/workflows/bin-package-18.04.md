# bin-package-18.04

**File:** `.github/workflows/bin-package-18.04.yaml`
**Trigger:** Called by other workflows (`workflow_call`)

## Purpose

Builds a single runtime package using an **Ubuntu 18.04** container. This is identical to `bin-package` but forces the older Ubuntu version for packages that require it (e.g. `mdadm`).

## Inputs

| Input | Required | Description |
|-------|----------|-------------|
| `package` | Yes | Name of the package to build |

## Secrets

| Secret | Description |
|--------|-------------|
| `token` | Hub JWT token for publishing flists |

## Steps

1. **Checkout** the repository
2. **Set tag**: Determines the version reference — uses git tag name for tag pushes, or short SHA for branch pushes
3. **Setup basesystem**: Runs `bins/bins-extra.sh --package basesystem` to install build prerequisites
4. **Build package**: Runs `bins/bins-extra.sh --package <package>` to compile the binary
5. **Publish flist**: Uploads the built binary as an flist to `tf-autobuilder/<package-name>.flist` on the hub
6. **Tagging**: On `main`, `testdeploy`, or version tags (`v*`), creates a tag link `<reference>/<package>.flist` pointing to the published flist
