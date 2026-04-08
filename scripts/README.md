# Helper scripts

| Script | Purpose |
|--------|---------|
| [`docs-upstream-pr-candidates.sh`](docs-upstream-pr-candidates.sh) | Lists merged PRs on **`grafana/tempo`** and (when run in GitHub Actions on **this fork**) opens a weekly **issue** for **`/docs-pr-check`** triage. Run locally without `GITHUB_REPOSITORY` to preview the table. **Note:** upstream PRs are queried **without** `GITHUB_TOKEN` because the Actions token is scoped to this repo only; using it for `grafana/tempo` would fail and previously looked like “no merges” when errors were swallowed. |

Used by [`.github/workflows/docs-upstream-pr-candidates.yml`](../.github/workflows/docs-upstream-pr-candidates.yml).
