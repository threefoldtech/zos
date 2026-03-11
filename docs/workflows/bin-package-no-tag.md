# bin-package-no-tag

**File:** `.github/workflows/bin-package-no-tag.yaml`
**Trigger:** Called by other workflows (`workflow_call`)

## Purpose

Builds a single runtime package using Ubuntu 20.04 but **never tags** the built binary. This means any package built with this workflow never becomes part of a ZOS installation release. Used for packages like `qsfs` and `traefik` that are published independently.

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
2. **Setup basesystem**: Installs build prerequisites via `bins/bins-extra.sh --package basesystem`
3. **Build package**: Compiles the binary via `bins/bins-extra.sh --package <package>`
4. **Publish flist**: Uploads the built binary as an flist to `tf-autobuilder/<package-name>.flist` on the hub

No tagging step is performed — the flist is published but not linked to any release version.
