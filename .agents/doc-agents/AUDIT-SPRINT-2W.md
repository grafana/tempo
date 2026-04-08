# Two-week docs vs code audit sprint

**Goal:** Complete a **first-pass code + doc review** in about **two weeks**, with emphasis on areas that **Tempo 3** changes most—and that readers hit first.

**Assumption:** Roughly **half-time focus** on this sprint (adjust blocks if you have more or less capacity).

Update the **Status** checkboxes as you go. If you slip a day, move the block—**priority order** below matters more than the exact calendar day.

---

## Why this order (Tempo 3)

**Tempo 3 introduces a major architecture break** from earlier versions. Readers upgrading or adopting fresh will go to:

1. **Set up for tracing** — how to get data in and Tempo running.  
2. **Configuration** — how the new system is wired and tuned.  
3. **Operations** — how to run, scale, and operate it.

Those three areas need to be **accurate against current code and the v3 model** before you polish less-trafficked sections.

**Secondary (still important, after P0):** **Troubleshooting** — high value when something breaks, but it assumes the primary paths above are right; fix misleading setup/config pointers here after P0 is solid.

Everything else (introduction, TraceQL deep dives, metrics-from-traces, API reference, release notes) can follow or fill buffer days.

---

## Before day 1 (30 minutes)

- [ ] Confirm fork is synced with `grafana/tempo` (or note how far behind you are)—prefer a branch or tag that reflects **Tempo 3** work if docs are being updated for that release.
- [ ] Link check is clean for `docs/sources/tempo` (run `./docs/docs-assistant/run-local-checks.sh` with `DOCS_DIR=docs/sources/tempo` or full tree).
- [ ] Open [`shared/docs-context-guide.md`](shared/docs-context-guide.md); use it to map claims → code. For v3, add notes on **what changed architecturally** as you find mismatches.
- [ ] Optional: run **`/docs-pr-check`** on PRs that touch `modules/`, `cmd/`, or `docs/` in the last few weeks and save the output.

---

## Week 1 — P0: set up for tracing + configuration

| Day | Focus (`docs/sources/tempo/`) | What to do | Status |
|-----|-------------------------------|------------|--------|
| **1** | **Set up for tracing** — overview, instrument-send, collectors entry (`set-up-for-tracing/`, `instrument-send/`, collector `_index` pages) | Map to **distributor / ingest path** in code; flag Alloy, OTel, tail-sampling gaps for v3. | [ ] |
| **2** | **Set up for tracing** — collectors depth (Grafana Alloy, OTel, **tail-sampling**) | Align policies and examples with current collector and Tempo receiver behavior. | [ ] |
| **3** | **Set up for tracing** — **setup-tempo** (deploy: local, Kubernetes, Helm, operator as applicable) | Deployment stories must match **how v3 is packaged and run**; note Helm/jsonnet pins. | [ ] |
| **4** | **Configuration** — overview, tenant IDs, storage/network slices (`configuration/_index.md`, hosted-storage, network, high-traffic pages) | Cross-check options vs code; **manifest** remains engineering-owned unless you see clear drift. | [ ] |
| **5** | **Configuration** — remaining pages you use in practice (parquet, polling, usage, etc.) | Finish configuration pass; list any option renames or removals for v3. | [ ] |

**End of week 1:** Short **findings list** (file → fix or issue). Run **`links.py --grafana-hugo`** on subtrees you changed.

---

## Week 2 — P0: operations, then P1: troubleshooting, then other

| Day | Focus | What to do | Status |
|-----|--------|------------|--------|
| **6** | **Operations** — CLI, caching, auth, scaling, monitoring entry (`operations/`) | Match `cmd/`, `modules/frontend`, compactor/querier paths; **ops runbooks** are P0 for v3 ops. | [ ] |
| **7** | **Operations** — advanced topics (multitenancy, overrides, dedicated columns, etc. as needed) | Close gaps in the operations tree that v3 touches. | [ ] |
| **8** | **Troubleshooting** (`troubleshooting/`) — **secondary priority** | Fix wrong “how to set up” pointers; align symptoms with **current** failure modes and dashboards. | [ ] |
| **9** | **Solutions-with-traces** (optional tie-in) + **gaps** | Ensure cross-links from troubleshooting to **set up / config / ops** are correct. | [ ] |
| **10** | **Buffer** — TraceQL overview, metrics-from-traces overview, or **API docs** spot-check | Use for whatever P0 left unfinished, or start secondary depth. | [ ] |

**Buffer days (11–14):** Deeper **TraceQL**, **metrics-from-traces**, **api_docs**, **introduction**, **release notes** (link hygiene only for old notes), **community**, **Helm get-started** pages.

---

## Every active day (15 minutes)

1. **`python3 docs/docs-assistant/links.py --grafana-hugo`** on the subtree you edited (or full `DOCS_DIR=docs/sources/tempo` at end of day).
2. Update **`context.yaml`** if v3 introduces new user-facing surfaces worth tracking.

---

## PR alignment (parallel)

- **Twice in the sprint:** **`/docs-pr-check`** on recent merges touching **set up, configuration, operations, or troubleshooting** paths.
- Prioritize PRs that land **code** changes for **Tempo 3** architecture.

---

## Definition of done (end of two weeks)

- [ ] **Set up for tracing**, **configuration**, and **operations** rows are checked **or** explicitly deferred with a ticket (these are **P0**).
- [ ] **Troubleshooting** pass completed **or** ticketed ( **P1** ).
- [ ] Findings are fixed, filed, or marked acceptable.
- [ ] **`links.py --grafana-hugo`** on `docs/sources/tempo` exits **0** after edits.

---

## If you run out of time

**Must not slip (P0):** `set-up-for-tracing/` (at least instrument → one full deploy path), `configuration/` (overview + the options readers set day one), `operations/` (CLI + one operational runbook depth).

**Do next (P1):** `troubleshooting/` (especially pages that send readers back to setup or config).

**Can slip to follow-up:** Introduction polish, deep TraceQL, metrics-from-traces details, API edge cases, old release notes beyond links, community.

---

## Related docs

- [`README.md`](README.md) — skills and Docs Assistant overview  
- [`README-LOCAL-CHECKS.md`](README-LOCAL-CHECKS.md) — link checks, cron, long-term plan  
