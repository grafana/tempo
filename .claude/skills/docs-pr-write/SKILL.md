---
name: docs-pr-write
description: Write or update Tempo docs for user-facing PR changes identified by docs-pr-check
allowed-tools: Bash Read Grep Write
---

# PR docs writer (execution phase)

For a prioritized PR list from `docs-pr-check`, create or update the required documentation pages.

This skill is for documentation execution only. Do not generate release notes.

## Usage

Invoke with `/docs-pr-write`.

Provide:
- PR numbers to process (recommended: only `Docs needed` and `Docs update needed` rows from `docs-pr-check`)
- The target Tempo version/branch context (for example `main`, `release-2.10`, or `3.0-docs`)

If the PR list is missing, ask for the output table from `docs-pr-check`.

## Inputs

Expected handoff from `docs-pr-check`:
- PR number
- Classification
- Notes about gaps
- Suggested target docs files, if known

## Steps to Perform

### 1. Confirm scope and order

1. Process only:
   - `Docs needed`
   - `Docs update needed`
2. Work in user-impact priority order:
   - Breaking changes/migrations
   - New `configuration` and API behavior
   - New query syntax and user workflows
   - Lower-risk clarifications

### 2. Reconstruct capability from each PR

For each PR:

```bash
gh pr view XXXX --repo grafana/tempo --json title,body,files,labels
```

Extract:
- What users can do now
- What changed in behavior
- New configuration fields/flags/endpoints/query syntax
- Version constraints and compatibility notes

### 3. Pick the canonical docs target

Prefer updating existing docs over creating new pages.

Use this order:
1. Existing page that already covers the topic
2. Existing related section where users already look
3. New page only if no suitable home exists

When uncertain between two pages, choose the one closest to user workflow and cross-link the other.

### 4. Write concise, task-oriented content

For each required change:
- Explain what changed in user terms
- Add when/why to use it
- Include one concrete, runnable example — match the format to the change type (curl for API endpoints, YAML for config, TraceQL for query syntax, shell command for CLI)
- **Required:** Note the minimum Tempo or storage format version. If unclear, flag as an open item rather than omitting.
- Call out default values when documenting configuration options

Keep content concise and avoid duplicating large reference material.

### 5. Validate claims against code

Do not rely only on PR description text.

Use the PR's changed file list as your guide to where to look. Only search within the `grafana/tempo` repository — do not search across other Grafana repositories.

For each claim you document, verify in the file type most likely to be authoritative:

- **Go files**: exact field name spelling in struct or constant, default value (`RegisterFlagsAndApplyDefaults`), accepted enum values
- **Proto files**: API endpoint paths, request/response field names
- **Helm/YAML files**: configuration keys and default values in deployment examples

Correct docs if code and PR text differ.

### 6. Link integration

Add links where users need them:
- From related docs sections to canonical page
- From release notes entries to canonical docs, if release notes are already being edited in the same task

Use consistent, clear link text (for example `documentation` when requested).

### 7. Final QA pass

Before returning:
- Confirm each PR in scope now has either updated docs or a justified blocker
- Check internal links and section anchors
- Keep style aligned with existing Tempo docs pages
- Keep language action-oriented and concise

### 8. Hand off verification items to the user

After completing all doc changes, output a "Before you submit" block. Select only the checklist sections from `.agents/doc-agents/shared/verification-checklist.md` that are relevant to what you just documented:

- Configuration option added or changed → include **Codebase Verification** and **Configuration Reference Check**
- New feature, query syntax, or API change → include **Version Compatibility**
- Style or wording edit only → omit code verification items entirely

Present the selected items as a short checklist for the user to complete before submitting the PR. Do not attempt to complete these items yourself.

## Return Format

Return:

1. **Files changed** (path list)
2. **PR-to-doc mapping**:
   - PR
   - what was documented
   - where it was documented
3. **Open items**:
   - uncertain claims needing engineering confirmation
   - deferred follow-up docs work

## Reference

- Triage skill: `.claude/skills/docs-pr-check/SKILL.md`
- Repo orientation: `.agents/doc-agents/shared/docs-context-guide.md` — code-to-docs mapping, key file paths, and Tempo doc conventions
- Workflow detail: `.agents/doc-agents/shared/release-notes-workflow.md`
- Verification checklist (Step 8 source): `.agents/doc-agents/shared/verification-checklist.md`
