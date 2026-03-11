# CodeQL Analysis

**File:** `.github/workflows/codeql-analysis.yml`
**Trigger:**
- Push to `main`
- Pull requests targeting `main`
- Scheduled: every Wednesday at 11:00 UTC

## Purpose

Runs GitHub's CodeQL static analysis tool on the Go codebase to find security vulnerabilities and code quality issues.

## Steps

1. **Checkout** the repository (with depth 2 for PR head detection)
2. **Initialize CodeQL** for Go language scanning
3. **Autobuild** — CodeQL automatically detects and builds the Go project
4. **Perform analysis** — runs the CodeQL queries and reports findings
