---
name: docs-workflow
description: End-to-end workflow for PR documentation — check, write, review. Use at any stage of documenting PR changes.
allowed-tools: Bash Read Grep Write
---

# PR documentation workflow

A three-step pipeline for documenting PR changes. Each step uses an existing skill; this workflow ties them together.

## Usage

You can use this workflow at any stage — before creating a docs PR, while drafting, or after PRs have shipped. It works the same way regardless of timing.

Invoke with `/docs-workflow`.

Provide:
- One or more PR numbers (from grafana/tempo)
- Target branch or version context (for example, `main`, `release-2.10`)


## Before you begin

Read `.agents/doc-agents/shared/docs-context-guide.md` for repo orientation: code-to-docs mapping, key file paths, and Tempo conventions.

## Steps

### 1. Check — identify documentation gaps

Run `.claude/skills/docs-pr-check/SKILL.md`.

**Input:** PR list from user.

**Output:** Classification table and prioritized gap summary. Each PR is classified as: docs present, docs needed, docs update needed, or no docs required.

Present the table to the user. Confirm which PRs to proceed with before moving to step 2.

### 2. Write — create or update documentation

Run `.claude/skills/docs-pr-write/SKILL.md`.

**Input:** Only PRs classified as "docs needed" or "docs update needed" from step 1, plus gap notes and suggested target files.

**Output:** Updated doc files, PR-to-doc mapping, and any open items needing engineering clarification.

Present the list of changed files. Ask the user to review before continuing to step 3.

### 3. Review — quality check the new content

Run `.claude/skills/docs-review/SKILL.md` on the files changed in step 2.

**Input:** File paths from step 2, plus any uncertain claims flagged during writing.

**Output:** Review report covering style guide compliance, frontmatter, links, and accuracy.

Present findings. After addressing review feedback, ask: _"Would you like me to create a PR, or would you prefer to review the changes locally first?"_

## Handoff contract

Each step passes forward:

| From | To | What passes |
|------|------|-------------|
| Step 1 → Step 2 | PR number, classification, gap notes, suggested target files |
| Step 2 → Step 3 | Changed file paths, open items or uncertain claims |

## When to use this vs. the writer-agent

| Task | Use |
|------|-----|
| PRs shipped and need docs | This workflow (`/docs-workflow`) |
| New feature or product docs from scratch | Writer-agent (`.agents/doc-agents/writers/writer-agent.md`) |
| General documentation work not tied to PRs | Writer-agent |

## Reference

- Repo orientation: `.agents/doc-agents/shared/docs-context-guide.md`
- Style guide: `.agents/doc-agents/shared/style-guide.md`
- Verification checklist: `.agents/doc-agents/shared/verification-checklist.md`
