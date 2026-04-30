# Cursor Automations × Tempo docs workflow

This guide wires **Cursor Automations** (cloud agents) to the same pipeline your local skills define: **`docs-pr-check` → `docs-pr-write` → `docs-review`**, orchestrated by **`docs-workflow`**.

**Official entry points**

- Create / manage automations: [cursor.com/automations](https://cursor.com/automations)
- Templates: [Cursor marketplace — Automations](https://cursor.com/marketplace#automations)
- Docs hub: [Cloud agent — Automations](https://cursor.com/docs/cloud-agent/automations) (UI may evolve; use the Automations page for the latest fields)

---

## 1. What Automations can and cannot replace

| Piece | Local skill | In Automations |
|-------|-------------|----------------|
| Triage merged Tempo PRs (heuristic) | `scripts/docs-pr-triage-local.sh` | Optional: keep as **cron** on your machine, or duplicate intent with a **scheduled** automation + `gh` / GitHub API |
| Deep gap analysis | `/docs-pr-check` | **Yes** — give the agent the same steps (PR list → `gh pr view` → classify → search `docs/sources/tempo`) |
| Writing docs | `/docs-pr-write` | **Yes**, but prefer **human confirmation** before opening a PR (see §4) |
| Quality pass | `/docs-review` | **Yes** — ideal for **PRs that only change `docs/**`** |

Automations run in a **cloud sandbox** with your instructions and configured **MCPs**. They do not read your local Cursor “skills” UI automatically — paste the **workflow + file paths** into the automation prompt (§5), or keep skill bodies short and link to this repo.

---

## 2. Prerequisites

1. **Repository**: Connect **`tempo-doc-work`** (your fork) to Cursor / the automation’s target repo.
2. **GitHub**: For Tempo PR data, the agent needs either:
   - **`gh` in the sandbox** with auth, or  
   - **GitHub MCP** / REST with a token that can read **`grafana/tempo`** PRs (and write issues/PRs on your fork if you want it to open PRs).
3. **Optional**: **Ripgrep**-style search in-repo is only available if the automation environment exposes it; keyword search over `docs/sources/tempo` may use `grep`/`rg` per agent capabilities.

---

## 3. Recommended triggers (pick one or combine)

| Goal | Trigger | Notes |
|------|---------|--------|
| Weekly Tempo triage | **Schedule** (e.g. Mon/Wed) | Same cadence as local `scripts/docs-pr-triage-local.sh` if you use it; output can mirror that script or run full `/docs-pr-check` on the PR list |
| Docs PR opened/updated | **GitHub: PR to `tempo-doc-work`** | Filter path **`docs/**`** — run **`docs-review`** on the diff |
| After you paste PR numbers | **Manual / webhook** | Custom webhook or Slack → automation with PR list in payload |

Start with **one scheduled “check-only”** automation and **one PR-triggered “review-only”** automation before chaining write steps.

---

## 4. Safety: split the pipeline across automations

**Recommended**

1. **Automation A — “Docs check”** (schedule): Input = time window or fixed PR list. Output = **classification table + gaps** (no writes). Posts to GitHub issue or Slack.
2. **Automation B — “Docs review”** (on PR): Input = changed files under `docs/sources/tempo`. Runs **`docs-review`**; comment on PR or fail a check.
3. **Automation C — “Docs write”** (manual approval): Trigger only from a **labeled issue**, **Slack approval**, or **workflow_dispatch**-style webhook — runs **`docs-pr-write`** and opens a **draft PR** for human merge.

Avoid a single automation that **writes production docs and merges** without review unless your org explicitly wants that.

---

## 5. Prompt skeleton (paste into the automation instructions)

Adapt the repo name and branches. Point the agent at these files **in the cloned repo**:

| Skill | Path in `tempo-doc-work` |
|-------|---------------------------|
| Workflow | `.claude/skills/docs-workflow/SKILL.md` |
| Check | `.claude/skills/docs-pr-check/SKILL.md` |
| Write | `.claude/skills/docs-pr-write/SKILL.md` |
| Review | `.claude/skills/docs-review/SKILL.md` |
| Orientation | `.claude/skills/shared/docs-context-guide.md` |
| Style | `.claude/skills/shared/style-guide.md` |

**Example system instructions**

```text
You are a technical documentation agent for the tempo-doc-work repository (Tempo product docs).

Repository layout: shipped docs live under docs/sources/tempo/. Always read:
- .claude/skills/shared/docs-context-guide.md
- .claude/skills/docs-workflow/SKILL.md for the three-phase pipeline

When asked to assess Tempo PRs (grafana/tempo):
1. Follow .claude/skills/docs-pr-check/SKILL.md: for each PR number, use gh pr view (or GitHub API) for title, body, files, labels. Classify: Docs present | Docs needed | Docs update needed | No docs required. Search docs/sources/tempo for feature names. Output a markdown table + prioritized gap list.

When asked to write docs:
2. Follow .claude/skills/docs-pr-write/SKILL.md only for PRs classified as Docs needed or Docs update needed. Edit only docs under docs/sources/tempo/ (and related frontmatter). Do not generate release notes.

When asked to review doc changes:
3. Follow .claude/skills/docs-review/SKILL.md on the listed paths. Run Vale if available; otherwise note Vale was skipped.

Constraints:
- Default Tempo repo for PRs: grafana/tempo.
- Docs repo: this clone (tempo-doc-work). Open a draft PR for doc changes; do not merge without human approval.
- If gh or GitHub access is missing, state what token or MCP is needed and stop.
```

---

## 6. MCPs and integrations

- **GitHub MCP** (if enabled in your workspace): use for PR files, comments, and opening draft PRs on **`knylander-grafana/tempo-doc-work`**.
- **Slack / Linear**: optional for notifications from Automation A.
- **Memory tool** (if offered): use to store “last triage PR set” or style preferences — optional.

---

## 7. Operational checklist

- [ ] Connect **`tempo-doc-work`** as the automation repository.
- [ ] Confirm **read access to `grafana/tempo`** PRs via `gh` or PAT.
- [ ] Create **Automation A** (schedule) with the prompt in §5 scoped to **step 1 only**.
- [ ] Create **Automation B** (PR trigger, `docs/**`) scoped to **step 3 only**.
- [ ] Add **Automation C** only after A/B are stable; require **draft PR** + human merge.
- [ ] Re-read Cursor’s Automations UI for **model selection**, **allowed commands**, and **secrets** naming — set least privilege on tokens.

---

## 8. Related local tooling

- Heuristic merged-PR triage (cron): `scripts/docs-pr-triage-local.sh` + `scripts/docs-pr-classify.jq` — see `scripts/README.md`.

This document is descriptive only; Cursor’s product UI is the source of truth for triggers, limits, and pricing.
