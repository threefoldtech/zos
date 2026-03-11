# Publish (Release)

**File:** `.github/workflows/publish.yaml`
**Trigger:** Push to any branch, or version tags (`v*`)

## Purpose

Builds all ZOS Go binaries (the core daemons) and publishes them as a single flist. This is the main release workflow for ZOS itself (as opposed to `bins.yaml` which handles runtime dependencies).

## Steps

1. **Set up Go 1.23**
2. **Checkout** the repository
3. **Build binaries**: Runs `make` in the `cmds/` directory to compile all ZOS daemons
4. **Set tag**: Determines version reference — tag name for releases, short SHA for branches
5. **Set version**: Generates a timestamped version like `v<YYMMDD.HHMMSS.0>`
6. **Collect files**: Runs `scripts/collect.sh` to gather all binaries and zinit configs into an archive
7. **Publish flist**: Uploads `zos:<version>.flist` to `tf-autobuilder`
8. **Tagging**: On `main`, `testdeploy`, or version tags, creates a tag link `<reference>/zos.flist`
9. **Cross tagging (development)**: On `main` or `testdeploy`, cross-tags the build to `tf-zos/development`, making it the latest development release

## Release Flow

```
Push to main
  → Build all daemons
  → Publish as zos:<version>.flist
  → Tag as <short-sha>/zos.flist
  → Cross-tag to tf-zos/development (devnet)

Push tag v*
  → Build all daemons
  → Publish as zos:<version>.flist
  → Tag as <tag>/zos.flist

Manual grid-deploy workflow
  → Links tf-zos/zos:<grid>-3:latest.flist → tf-autobuilder/zos:<version>.flist
```
