# Tempo documentation tooling (overview)

This folder holds **human and agent guidance** for writing Tempo docs (`writer-agent.md`, `shared/`). **Mechanical tooling** for link checking, coverage maps, and optional scheduling lives under **`docs/docs-assistant/`** and **`.claude/skills/`**. Together they cover:

| Layer | What it does | Where it lives |
|-------|----------------|----------------|
| **Skills** | PR triage, drafting, review in Cursor/Claude (interactive) | [`.claude/skills/`](../.claude/skills/README.md) |
| **Docs Assistant** | `context.yaml` coverage map, prompts, Markdown link checker | [`docs/docs-assistant/`](../docs/docs-assistant/) |
| **Repo & sync** | Your fork tracks `grafana/tempo` (optional GitHub Action) | `.github/workflows/` (if enabled) |

The top-level [`.agents/README.md`](../README.md) explains **skills and writer workflows** in detail. This document ties in **Docs Assistant**, **link checks**, and **local automation** so nothing is mysterious when you work in `tempo-doc-work`.

**Time-boxed audit:** For a **~two-week** code + doc review plan and daily checklist, use [`AUDIT-SPRINT-2W.md`](AUDIT-SPRINT-2W.md). It prioritizes **set up for tracing**, **configuration**, and **operations** (Tempo 3 architecture), then **troubleshooting**.

---

## Skills (interactive, PR-driven)

Use these when you have PR numbers or changed files to review:

- **`/docs-workflow`** — `docs-pr-check` → `docs-pr-write` → `docs-review` in sequence.
- **`/docs-pr-check`**, **`/docs-pr-write`**, **`/docs-review`** — each step alone.

They read GitHub PRs, search `docs/sources/tempo/`, and apply [`shared/style-guide.md`](shared/style-guide.md) and related resources. They do **not** replace engineering processes (for example **generated configuration manifest** updates are owned by engineering releases).

---

## Docs Assistant (repo files, not a package)

**Docs Assistant** here means the **`docs/docs-assistant/`** directory: prompts, **`context.yaml`**, and **`links.py`**. It is **vendored** into this repo (same pattern as other Grafana docs repos). There is no separate `pip install`.

| File / area | Role |
|-------------|------|
| **`context.yaml`** | Coverage map: which article should reflect which user-facing behavior. Used when you (or an agent) align docs with code changes. Regenerate or edit when the doc tree shifts; see `prompts/generate-context.md` and `prompts/update-context.md`. |
| **`prompts/docs-assistant.md`** | Writing guide and Tempo-specific conventions for the assistant. |
| **`prompts/update-docs.md`** | Instructions for update-style tasks (diff vs coverage). |
| **`links.py`** | Validates Markdown links in a **directory** of sources. |

**GitHub Actions** from the upstream Docs Assistant installer (optional PR comment workflows) are **not required** for local work. You can use everything here without enabling those workflows.

---

## Link checking (`links.py` and Grafana Hugo)

Tempo docs use **Grafana Hugo** linking: paths like `/docs/tempo/...` and `https://grafana.com/...` are resolved by the **website** build, not as plain files in this clone.

The stock **`links.py`** “strict” mode treats `/docs/...` as errors. For **Tempo sources**, always run with **`--grafana-hugo`** so site paths and shortcodes are skipped and you only see **real** relative issues (broken paths, missing targets next to the page).

```bash
# From repo root — example: one section
python3 docs/docs-assistant/links.py --grafana-hugo docs/sources/tempo/configuration
```

**`run-local-checks.sh`** passes **`--grafana-hugo` by default**. Use **`LINK_CHECK_STRICT=1`** only if you intentionally want the old strict behavior.

Details: [`docs/docs-assistant/README.md`](../docs/docs-assistant/README.md) (section “Check links locally”) and [`README-LOCAL-CHECKS.md`](README-LOCAL-CHECKS.md) (cron, logs, troubleshooting).

---

## Local scheduling (cron / launchd)

Optional: run the same link check on a schedule and inspect logs before proposing anything upstream.

- **`docs/docs-assistant/run-local-checks.sh`** — wraps `links.py --grafana-hugo` and optional `DOCS_DIR`.
- **[`README-LOCAL-CHECKS.md`](README-LOCAL-CHECKS.md)** — how to run checks manually, schedule **cron** / **launchd**, read logs, and propose the same step upstream.

Agent skills (`/docs-workflow`, etc.) **cannot** be invoked from cron; they are **interactive**. Cron is for **scriptable** checks only.

---

## Fork and upstream

- **`tempo-doc-work`** is a **fork** used for writing and validation. You may use a **sync workflow** so `main` tracks `grafana/tempo` without manual merges.
- **Upstream** does not need your Docs Assistant or local cron; if you propose automation later, bring the **same commands** (`python3 ... links.py --grafana-hugo …`) so behavior matches what you already tested.

---

## How the pieces fit (typical flow)

1. **Sync** your fork when you start a session (or rely on scheduled sync).
2. **Triage incoming work** with **`/docs-pr-check`** on merged PRs you care about (or a chunked **audit** of a `docs/sources/tempo/...` subtree).
3. **Draft or fix** with **`/docs-pr-write`** or **`writer-agent.md`**, using **`context.yaml`** to see what each page should cover.
4. **Review** with **`/docs-review`** and [`shared/verification-checklist.md`](shared/verification-checklist.md).
5. **Before commit** (or on a schedule), run **`links.py --grafana-hugo`** on the subtree you touched or on `docs/sources/tempo` if you want a broader pass.

---

## Shared resources (this folder)

| Path | Use |
|------|-----|
| [`shared/README.md`](shared/README.md) | Index of style guide, context guide, release notes workflow, checklists. |
| [`shared/docs-context-guide.md`](shared/docs-context-guide.md) | Where code maps to docs in this repo. |
| [`shared/style-guide.md`](shared/style-guide.md) | Grafana style and templates. |
| [`writers/writer-agent.md`](writers/writer-agent.md) | Long-form doc workflow from scratch. |

---

## Quick reference

| I want to… | Use |
|------------|-----|
| Triage PRs for doc gaps | `/docs-pr-check` |
| Write or update docs for PRs | `/docs-pr-write` or `/docs-workflow` |
| Review drafted docs | `/docs-review` |
| Check Markdown links (Tempo Hugo) | `python3 docs/docs-assistant/links.py --grafana-hugo <path-to-dir>` |
| Run link check with defaults | `./docs/docs-assistant/run-local-checks.sh` |
| See coverage map | `docs/docs-assistant/context.yaml` |
| Schedule link checks locally | [`README-LOCAL-CHECKS.md`](README-LOCAL-CHECKS.md) |
