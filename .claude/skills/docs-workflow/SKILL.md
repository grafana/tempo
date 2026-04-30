---
name: docs-workflow
description: End-to-end workflow for PR documentation — check, write, review, and optionally validate screenshots. Use when the user wants to document one or more PRs from start to finish, run the full docs pipeline, or asks for a complete documentation pass on PRs.
metadata:
  use_case: write-or-update, quality-review
  workflow: create, evaluate
---

# PR documentation workflow

A three-step pipeline for documenting PR changes: assess gaps, write docs, review quality. Each step uses an existing skill; this workflow ties them together.

## Before you begin

1. Load local context per [`../shared/load-context.md`](../shared/load-context.md). The sub-skills inherit this context.
2. Get one or more PR numbers and target branch context from the user.

## Steps

### 1. Check — identify documentation gaps

Run [`../docs-pr-check/SKILL.md`](../docs-pr-check/SKILL.md).

**Input:** PR list from user.
**Output:** Classification table and prioritized gap summary.

Present the table to the user. Confirm which PRs to proceed with before moving to step 2.

### 2. Write — create or update documentation

Run [`../docs-pr-write/SKILL.md`](../docs-pr-write/SKILL.md).

**Input:** PRs classified as "docs needed" or "docs update needed" from step 1, plus gap notes and suggested target files.
**Output:** Updated doc files, PR-to-doc mapping, and open items.

Present the list of changed files. Ask the user to review before continuing to step 3.

### 3. Review — quality check the new content

Run [`../docs-review/SKILL.md`](../docs-review/SKILL.md) on the files changed in step 2.

**Input:** File paths from step 2, plus any uncertain claims flagged during writing.
**Output:** Review report covering style guide compliance, frontmatter, links, and accuracy.

Present findings. After addressing review feedback, ask: _"Would you like me to create a PR, or would you prefer to review the changes locally first?"_

### Optional: Screenshot validation

If step 1 produced a **Screenshot inventory** (PRs with UI changes affecting pages that have screenshots), offer to validate them. If yes, run `screenshot-check` (if available), passing the inventory table from step 1 directly. The user must provide a live URL.

## Handoff contract

| From | To | What passes |
|------|------|-------------|
| Step 1 → Step 2 | PR number, classification, gap notes, suggested target files |
| Step 2 → Step 3 | Changed file paths, open items or uncertain claims |
| Step 1 → Screenshot check | Screenshot inventory: PR numbers, affected doc pages, screenshot count, image references |

## When to use this vs. individual skills

| Task | Use |
|------|-----|
| One or more PRs need docs end-to-end | This workflow |
| Just need to check if PRs have docs | `docs-pr-check` standalone |
| Already know what to write, have PR numbers | `docs-pr-write` standalone |
| Review existing doc changes | `docs-review` standalone |
| New docs from scratch, not tied to PRs | `docs-from-code` (if available) |

## Reference

- Repo orientation: `../shared/docs-context-guide.md`
- Style guide: `../shared/style-guide.md`
- Verification checklist: `../shared/verification-checklist.md`
- Screenshot validation: `screenshot-check` (if available)
