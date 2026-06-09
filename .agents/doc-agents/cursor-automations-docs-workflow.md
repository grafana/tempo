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
| Triage merged Tempo PRs | `/docs-pr-check` | **Yes** — schedule an automation that runs the check skill (PR list → `gh pr view` → classify → search `docs/sources/tempo`) |
| Writing docs | `/docs-pr-write` | **Yes**, but prefer **human confirmation** before opening a PR (see [Safety: split the pipeline across automations](#4-safety-split-the-pipeline-across-automations)) |
| Quality pass | `/docs-review` | **Yes** — ideal for **PRs that only change `docs/**`** |

Automations run in a **cloud sandbox** with your instructions and configured **MCPs**. They do not read your local Cursor “skills” UI automatically — paste the **workflow + file paths** into the automation prompt ([Prompt skeleton](#5-prompt-skeleton-paste-into-the-automation-instructions)), or keep skill bodies short and link to this repo.

---

## 2. Prerequisites

1. **Repository**: Connect **your docs repository** (the clone where you maintain Tempo docs) to Cursor / the automation’s target repo. Substitute `YOUR_ORG/YOUR_REPO` from `docs/project-context.md`.
2. **GitHub**: For Tempo PR data, the agent needs either:
   - **`gh` in the sandbox** with auth, or
   - **GitHub MCP** / REST with a token that can read **`grafana/tempo`** PRs (and write issues/PRs on your docs repo if you want it to open PRs).
3. **Optional**: **Ripgrep**-style search in-repo is only available if the automation environment exposes it; keyword search over `docs/sources/tempo` may use `grep`/`rg` per agent capabilities.

---

## 3. Recommended triggers (pick one or combine)

| Goal | Trigger | Notes |
|------|---------|--------|
| Weekly Tempo triage | **Schedule** (e.g. Mon/Wed) | Run `/docs-pr-check` on recently merged PRs; output is a classification table plus gap summary |
| Docs PR opened/updated | **GitHub: PR to your docs repo** | Filter path **`docs/**`** — run **`docs-review`** on the diff |
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

| Skill | Path in your repo |
|-------|---------------------------|
| Workflow | `.claude/skills/docs-workflow/SKILL.md` |
| Check | `.claude/skills/docs-pr-check/SKILL.md` |
| Write | `.claude/skills/docs-pr-write/SKILL.md` |
| Review | `.claude/skills/docs-review/SKILL.md` |
| Orientation | `.claude/skills/shared/docs-context-guide.md` |
| Style | `.claude/skills/shared/style-guide.md` |

**Example system instructions**

```text
You are a technical documentation agent for this repository (Tempo product docs).

Repository layout: shipped docs live under docs/sources/tempo/. Always read:
- docs/project-context.md (repo-specific paths and conventions)
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
- Docs repo: this clone (YOUR_ORG/YOUR_REPO). Open a draft PR for doc changes; do not merge without human approval.
- If gh or GitHub access is missing, state what token or MCP is needed and stop.
```

---

## 6. MCPs and integrations

- **GitHub MCP** (if enabled in your workspace): use for PR files, comments, and opening draft PRs on **YOUR_ORG/YOUR_REPO**.
- **Slack / Linear**: optional for notifications from Automation A.
- **Memory tool** (if offered): use to store “last triage PR set” or style preferences — optional.

---

## 7. Operational checklist

- [ ] Connect **your docs repository** as the automation target.
- [ ] Confirm **read access to `grafana/tempo`** PRs via `gh` or PAT.
- [ ] Create **Automation A** (schedule) using the [Prompt skeleton](#5-prompt-skeleton-paste-into-the-automation-instructions), scoped to **check only** (docs-pr-check).
- [ ] Create **Automation B** (PR trigger, `docs/**`) using the same prompt skeleton, scoped to **review only** (docs-review).
- [ ] Add **Automation C** only after A/B are stable; require **draft PR** + human merge.
- [ ] Re-read Cursor’s Automations UI for **model selection**, **allowed commands**, and **secrets** naming — set least privilege on tokens.

This document is descriptive only; Cursor’s product UI is the source of truth for triggers, limits, and pricing.
