# Helper scripts

| File | Purpose |
|------|---------|
| [`docs-pr-triage-local.sh`](docs-pr-triage-local.sh) | **Local only** (cron / launchd — **not** GitHub Actions). Lists merged PRs on **`grafana/tempo`**, classifies docs status using [`docs-pr-classify.jq`](docs-pr-classify.jq) (heuristic approximation of `/docs-pr-check`), and opens an issue on **your fork** via `gh`. Use `--dry-run` to print markdown without creating an issue. |
| [`docs-pr-classify.jq`](docs-pr-classify.jq) | `jq` rules used by the script; each PR gets `classification`, `notes`, and **`priority`** (`high` / `medium` / `low`) for gap ordering. Edit to tune heuristics. |

**Environment:** `UPSTREAM_REPO` (default `grafana/tempo`), `SINCE_DAYS` (default `7`), `ISSUE_REPO` (default: `gh repo view` from the clone), **`DOCS_TEMPO_ROOT`** (default: `<repo>/docs/sources/tempo` for this repo). When `DOCS_TEMPO_ROOT` exists and **`rg`** (ripgrep) is on `PATH`, the script runs a lightweight keyword search over `*.md` there (roughly the `/docs-pr-check` “search `docs/sources/tempo`” step). Requires `gh` auth with permission to create issues on the fork.

**Accuracy vs `/docs-pr-check`:** The skill classifies from **meaning** (what changed, whether behavior is user-visible) and **reads shipped docs for completeness**. The script uses **labels**, **PR checklist + paths**, **title patterns**, **CI-only paths**, and **optional keyword hits**; it cannot match the skill fully without a human or model. **Run `/docs-pr-check`** on the copy-paste PR list for the authoritative pass.

**Schedule:** See [README-LOCAL-CHECKS.md](../.agents/doc-agents/README-LOCAL-CHECKS.md#scheduled-docs-pr-triage-monwed) for **Monday/Wednesday** `crontab` examples.
