# docs-pr-check: PR #5840 and #5669

## Classification Table

| PR | Title | Classification | Notes |
|----|-------|---------------|-------|
| #5840 | build(deps): bump cloud.google.com/go/storage from 1.56.0 to 1.57.1 | No docs required | Dependabot dependency bump (Go module). Labels: `dependencies`, `go`. Only changes `go.mod`, `go.sum`, and vendored library files. No user-facing behavior change. |
| #5669 | Bump anchore/sbom-action from 0.20.4 to 0.20.6 | No docs required | Dependabot dependency bump (GitHub Actions). Labels: `dependencies`, `github_actions`. Only changes `.github/workflows/sbom-report.yml`. CI/CD change with no user-visible impact. |

## Gap Summary

Neither PR requires documentation work.

Both PRs are automated Dependabot dependency upgrades:

- **#5840** bumps the `cloud.google.com/go/storage` Go module from v1.56.0 to v1.57.1. Changes are confined to `go.mod`, `go.sum`, and vendored files under `vendor/cloud.google.com/go/storage/`. There are no new configuration options, behavioral changes, or user-facing features introduced in the Tempo codebase itself.

- **#5669** bumps the `anchore/sbom-action` GitHub Actions workflow action from v0.20.4 to v0.20.6. The only changed file is `.github/workflows/sbom-report.yml`. This is a CI/CD-internal change with no impact on Tempo users or operators.

**Action required:** None. No documentation needs to be created or updated for either PR.
