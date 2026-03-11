# Grid Deploy

**File:** `.github/workflows/grid-deploy.yaml`
**Trigger:** Manual dispatch (`workflow_dispatch`)

## Purpose

Deploys a specific ZOS version to a target grid environment (qa, testing, or production) by creating a cross-link on the hub. This effectively makes the specified version the "latest" for that grid.

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `grid` | Yes | `qa` | Target grid environment (`qa`, `testing`, or `production`) |
| `version` | Yes | — | Version string to deploy (e.g. the flist version tag) |

## Steps

1. **Symlink flist**: Creates a cross-link so that `tf-zos/zos:<grid>-3:latest.flist` points to `tf-autobuilder/zos:<version>.flist`

This is the mechanism used to promote a tested build to a specific grid environment.
