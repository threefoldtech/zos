# bin-package

**File:** `.github/workflows/bin-package.yaml`
**Trigger:** Called by other workflows (`workflow_call`)

## Purpose

Builds a single runtime package using Ubuntu 20.04. The built binary is always published to `tf-autobuilder` and then tagged with the release version or the short SHA of the commit head (on main branch). This is the standard build workflow invoked by `bins.yaml` for most packages.

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
2. **Set tag**: Determines the version reference — uses git tag name for tag pushes (e.g. `v1.2.3`), or short SHA for branch pushes
3. **Setup basesystem**: Runs `apt update` then `bins/bins-extra.sh --package basesystem`
4. **Build package**: Compiles the binary via `bins/bins-extra.sh --package <package>`
5. **Publish flist**: On `main`, `testdeploy`, or version tags, uploads to `tf-autobuilder/<package-name>.flist`
6. **Tagging**: On `main`, `testdeploy`, or version tags, creates a tag link `<reference>/<package>.flist` pointing to the published flist
