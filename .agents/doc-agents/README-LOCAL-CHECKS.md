# Local checks and scheduled runs (cron / launchd)

This README explains how to **run Markdown link checks on your machine**, optionally on a **schedule**, without GitHub Actions. The same commands are what you would later propose for **CI** on `grafana/tempo` if the team wants them. It also explains how that fits the broader plan: **chunked docs-vs-code audit**, then **ongoing PR-based maintenance** (see [Full docs vs code audit, then PR maintenance](#full-docs-vs-code-audit-then-pr-maintenance)).

**Related:** [`README.md`](README.md) (skills, Docs Assistant overview), [`docs/docs-assistant/README.md`](../../docs/docs-assistant/README.md) (Docs Assistant install and `--grafana-hugo`).

---

## Full docs vs code audit, then PR maintenance

**Local link checks** (`run-local-checks.sh` / `links.py --grafana-hugo`) are a **mechanical** guardrail: broken relative links and missing assets next to Markdown files. They do **not** prove that prose matches the latest code behavior. The larger plan has two parts: **(1) a chunked audit** of docs against the codebase, **(2) ongoing alignment** with merged PRs.

### Already in place (tooling)

| Piece | Role |
|-------|------|
| **Skills** (`/docs-pr-check`, `/docs-pr-write`, `/docs-review`, `/docs-workflow`) | Triage PRs for doc gaps, draft updates, review. Interactive — not runnable from cron. |
| **`docs/docs-assistant/context.yaml`** | Coverage map: which pages should reflect which behavior. Regenerate or edit when the doc tree changes. |
| **Fork sync** (optional GitHub Action) | Keeps `tempo-doc-work` close to `grafana/tempo`. |
| **This README** | Cron / launchd for **link** checks only. |

**Generated configuration manifest** (`make generate-manifest` / `manifest.md`) is owned by **engineering workflows**; treat drift there as an engineering/release concern unless you spot an obvious gap while writing.

### Tempo 3 and audit priority

**Tempo 3** is a major **architecture break** from earlier versions. For a docs-vs-code pass, treat these as **first** (they get the most visits and must match the new model): **set up for tracing**, **configuration**, **operations**. **Troubleshooting** is next—after P0 is solid, so symptom pages don’t send readers to outdated setup or config. A concrete two-week schedule with that order is in [`AUDIT-SPRINT-2W.md`](AUDIT-SPRINT-2W.md).

### What you still need to operationalize (process)

1. **Chunked audit backlog** — Divide `docs/sources/tempo/` (or your `context.yaml` sections) into **chunks** (by area: configuration, TraceQL, set-up, operations, …). Order by **risk** or **reader traffic**, not alphabetically. Work one chunk at a time. A concrete **two-week schedule** lives in [`AUDIT-SPRINT-2W.md`](AUDIT-SPRINT-2W.md).

2. **Tracking** — Use a board, issues, or a checklist so **which chunks are done** is visible and chunks don’t stall between sessions.

3. **Definition of done (per chunk)** — e.g. reviewed docs + spot-checked relevant code paths + fixed docs or filed follow-ups. “Audit complete” means every chunk has been through that once.

4. **PR maintenance cadence** — On a **schedule you actually use** (weekly, biweekly): run **`/docs-pr-check`** on **recent merged PRs** in areas you care about (narrow by time window or paths). Skills cover *how*; you still need *when* (calendar reminder, recurring task). **Optional automation on this fork only:** [`.github/workflows/docs-upstream-pr-candidates.yml`](../../.github/workflows/docs-upstream-pr-candidates.yml) runs weekly and opens an issue listing merged PRs on **`grafana/tempo`** (upstream PRs; **no** Actions on the Tempo repo). Use that issue as your PR list for **`/docs-pr-check`**. Enable **Actions** and **scheduled workflows** on the fork if jobs don’t run.

5. **Scope for PR triage** — Agree what counts as user-facing (e.g. `modules/`, `cmd/`, `pkg/traceql/`, public APIs) vs noise (tests-only, vendor-only). Keeps triage small.

6. **Link debt** — Fix or redirect **known bad relative links** so `links.py` stays **green** (or intentionally scoped). Otherwise cron and future CI always red, which erodes trust.

### Optional later

- **Upstream PR digest (tempo-doc-work)** — Workflow **`docs-upstream-pr-candidates.yml`** (see [`scripts/README.md`](../../scripts/README.md)): weekly issue with links to merged PRs on **`grafana/tempo`**. You still classify with **`/docs-pr-check`**; the workflow does not touch **`grafana/tempo`**.
- **Drift script** — e.g. merged PRs touching user-facing paths without `docs/` changes. Not required if **`/docs-pr-check`** runs regularly.
- **CI link check upstream** — Same `links.py --grafana-hugo` command you validated locally; only if the team adds it to **`grafana/tempo`** later.

### How local checks fit

Run **`run-local-checks.sh`** (or cron) **after** editing links or on a schedule for **regression detection** on relative links. Run **skills** when you need **semantic** alignment (PRs, features, audits). Neither replaces the other.

---

## What runs locally

| Script | What it does |
|--------|----------------|
| [`run-local-checks.sh`](../../docs/docs-assistant/run-local-checks.sh) | Calls [`links.py`](../../docs/docs-assistant/links.py) with **`--grafana-hugo`** on a docs directory (default: entire `docs/sources/`). |

**`--grafana-hugo`** is required for Tempo sources: paths like `/docs/tempo/...`, `/media/...`, and `ref:` are **skipped** (they resolve on the Grafana website build). Without it, the checker reports hundreds of false positives.

The wrapper also supports:

| Environment variable | Meaning |
|---------------------|---------|
| `DOCS_DIR` | Root to scan (default: `docs/sources/` from repo root). Example: `docs/sources/tempo/configuration` for one section. |
| `LINK_CHECK_STRICT=1` | Use **strict** `links.py` (flags `/docs/` as wrong). Only for repos that use relative links only—not typical for Tempo. |
| `PYTHON` | Python to use (default: `python3`). Set to an absolute path in cron if `python3` is not on cron’s minimal `PATH`. |

You can run `links.py` directly when debugging:

```bash
python3 docs/docs-assistant/links.py --grafana-hugo docs/sources/tempo/configuration
```

---

## Run checks manually (before you automate)

1. `cd` to the **repository root** (`tempo-doc-work`).
2. Ensure the script is executable once: `chmod +x docs/docs-assistant/run-local-checks.sh`
3. Run:

```bash
./docs/docs-assistant/run-local-checks.sh
```

4. **Exit code 0** — no link issues the checker can see. **Exit code 1** — see the printed `ERROR:` lines (broken relative paths, missing files next to the page, etc.).

Narrow the scope while iterating:

```bash
DOCS_DIR=docs/sources/tempo ./docs/docs-assistant/run-local-checks.sh
```

Fix or triage failures until the output matches what you expect for cron (you may still choose to allow a non-zero exit until legacy links are cleaned up).

---

## Schedule with cron (Linux / macOS)

Cron runs in a **minimal environment**: few environment variables, a short `PATH`, and no interactive shell. Use **absolute paths** for `cd`, the script, Python (if needed), and the log file.

### 1. Pick a log directory

Example (under your home directory, not committed to git):

```bash
mkdir -p ~/.cache/tempo-doc-work
```

### 2. Test the exact command line

Run the same line you will put in crontab, including redirection:

```bash
cd /ABS/PATH/TO/tempo-doc-work && ./docs/docs-assistant/run-local-checks.sh >> ~/.cache/tempo-doc-work/docs-checks.log 2>&1
```

Replace `/ABS/PATH/TO/tempo-doc-work` with your clone path. Open the log and confirm output and exit behavior.

### 3. Install a crontab entry

```bash
crontab -e
```

Example: **every Monday at 09:00** in the server’s local timezone:

```cron
0 9 * * 1 cd /ABS/PATH/TO/tempo-doc-work && ./docs/docs-assistant/run-local-checks.sh >> ~/.cache/tempo-doc-work/docs-checks.log 2>&1
```

Optional: set `DOCS_DIR` for a smaller weekly pass:

```cron
0 9 * * 1 cd /ABS/PATH/TO/tempo-doc-work && DOCS_DIR=docs/sources/tempo ./docs/docs-assistant/run-local-checks.sh >> ~/.cache/tempo-doc-work/docs-checks.log 2>&1
```

### 4. If cron cannot find `python3`

Get the interpreter path:

```bash
which python3
```

Then either add to the crontab line:

```cron
PYTHON=/usr/bin/python3
```

(adjust path) before the command, or export it in the same line:

```cron
0 9 * * 1 cd /ABS/PATH/TO/tempo-doc-work && PYTHON=/usr/local/bin/python3 ./docs/docs-assistant/run-local-checks.sh >> ~/.cache/tempo-doc-work/docs-checks.log 2>&1
```

---

## macOS: launchd instead of cron

On Apple Silicon and recent macOS, **`launchd`** often behaves more predictably than cron (sleep/wake, logging, environment). The idea is the same: run `cd … && ./docs/docs-assistant/run-local-checks.sh` and append stdout/stderr to a log file. Create a **plist** with `StartCalendarInterval` or `StartInterval`, set `WorkingDirectory` to the repo root, and load with `launchctl bootstrap` / `launchctl load` (see Apple’s documentation for your OS version).

---

## Reading logs and exit codes

| Result | Meaning |
|--------|--------|
| **Exit 0** | Checker found no issues in the scanned tree. |
| **Exit 1** | At least one `ERROR:` line; see log. |

Cron does not email you unless you configure mail; rely on **periodically reading the log** or a wrapper script that notifies you when the exit code is non-zero.

Rotate or truncate the log occasionally so files do not grow without bound:

```bash
# example: keep last 2000 lines
tail -n 2000 ~/.cache/tempo-doc-work/docs-checks.log > /tmp/docs-checks.log && mv /tmp/docs-checks.log ~/.cache/tempo-doc-work/docs-checks.log
```

---

## What does not belong in cron

These require an interactive editor or GitHub:

- **`/docs-workflow`**, **`/docs-pr-check`**, **`/docs-pr-write`**, **`/docs-review`** (skills under `.claude/skills/`)
- **`gh`**-based drift scripts unless you explicitly script auth and accept maintenance cost

Use cron only for **deterministic scripts** like `run-local-checks.sh`. When a report shows problems, open the repo and run skills or edit docs yourself.

---

## Proposing the same check upstream

After the local log looks stable:

1. Agree with the team on **scope** (`docs/sources` vs `docs/sources/tempo` only) and on **`--grafana-hugo`** for Tempo Hugo sources.
2. Add a GitHub Actions workflow with a step equivalent to:

   ```yaml
   - run: python3 docs/docs-assistant/links.py --grafana-hugo docs/sources
   ```

3. Prefer **`pull_request`** with `paths: docs/**` first; add **`schedule`** if the team wants periodic visibility.

---

## Troubleshooting

| Problem | What to try |
|--------|----------------|
| `python3: not found` in cron | Set `PYTHON` to the full path from `which python3`. |
| Huge error lists | Narrow `DOCS_DIR`; fix backlog; confirm `--grafana-hugo` is used (default in `run-local-checks.sh`). |
| Script not found | Use absolute path to `run-local-checks.sh` in cron, or `cd` to repo root first. |
| Permission denied | `chmod +x docs/docs-assistant/run-local-checks.sh` |

---

## See also

- [`README.md`](README.md) — full picture: skills, Docs Assistant, fork sync, and how pieces connect.
- [`docs/docs-assistant/LOCAL_SCHEDULER.md`](../../docs/docs-assistant/LOCAL_SCHEDULER.md) — stub that points here.
- [`docs/docs-assistant/links.py`](../../docs/docs-assistant/links.py) — checker implementation and `--grafana-hugo` behavior.
