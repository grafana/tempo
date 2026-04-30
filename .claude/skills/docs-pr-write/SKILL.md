---
name: docs-pr-write
description: >-
  Write or update product documentation for user-facing PR changes.
  Use when the user asks to write docs for a PR, document a PR,
  update existing docs based on a code change, process PRs flagged
  by docs-pr-check, or provides PR numbers or links and wants the
  corresponding documentation created or updated — even if they
  say "document these changes" without mentioning PRs explicitly.
  This skill writes and updates docs only; it does not triage PRs
  (use docs-pr-check), review existing docs (use docs-review), or
  generate release notes.
metadata:
  use_case: write-or-update
  workflow: create
---

# PR docs writer

Write or update documentation for user-facing PR changes. Do not generate release notes.

## Before you begin

1. Load local context per [`../shared/load-context.md`](../shared/load-context.md).
2. If the user mentions a release version or release notes, also read `../shared/release-notes-workflow.md` (Phases 1.5–1.75). Otherwise skip it — this skill works on any PR independently.

## Inputs

Either:

- **PR numbers directly** — triage inline using the criteria in [`../docs-pr-check/SKILL.md`](../docs-pr-check/SKILL.md) before writing.
- **A classified PR list from `docs-pr-check`** — with PR numbers, classifications, gap notes, and suggested target files. Use these classifications as-is.

## Steps

### 1. Confirm scope and order

Process only `Docs needed` and `Docs update needed` PRs. Work in user-impact order: breaking changes and migrations first, then new config/API behavior, then new syntax and workflows, then lower-risk clarifications.

### 2. Reconstruct capability from each PR

```bash
gh pr view XXXX --repo YOUR_ORG/YOUR_REPO --json title,body,files,labels
```

Extract: what users can now do, what changed in behavior, new config fields/flags/endpoints/syntax, and version constraints.

### 3. Pick the docs target

Prefer updating existing pages over creating new ones:

1. Existing page that already covers the topic
2. Related section where users already look
3. New page only if no suitable home exists

When torn between two pages, choose the one closest to the user workflow and cross-link the other.

### 4. Write concise, task-oriented content

- Explain what changed in user terms
- Add when/why to use it
- Include one concrete, runnable example matching the change type (HTTP for APIs, YAML for config, query snippets for query languages, shell for CLI)
- Note the minimum version when relevant — users on older versions hit confusing errors without this
- Call out default values for configuration options
- Add links from related sections to the canonical page, and from release notes entries if they're being edited in the same task

Match the persona level of the target page (refer to `../shared/personas.md` when uncertain). Follow `../shared/style-guide.md`.

### 5. Validate claims against code

Do not rely only on PR description text.

**When to load detail:** For any PR that adds or changes **technical** claims (behavior, APIs, config, UI copy, routes), read [`references/validate-claims.md`](references/validate-claims.md) and follow it. Skip this reference for **purely editorial** edits with no new technical assertions.

**Default approach:** Verify **PR-scoped** only (changed files + their import chain + paths from **project-context** validation tables when present). Do not scan the whole monorepo. Use [`../shared/load-context.md`](../shared/load-context.md) for where key paths live.

### 6. Finish

Before returning:

- Confirm each in-scope PR now has docs or a justified blocker
- Check internal links and section anchors
- Use `../shared/style-guide.md` as the primary style authority. If the target page sits in a section with similar pages (same topic area, same doc type), skim 1–2 of those pages to confirm tone and depth — but defer to the style guide when they conflict.
- Select relevant sections from `../shared/verification-checklist.md` based on what you documented (config changes → **Codebase Verification** + **Configuration Reference Check**; new features/APIs → **Version Compatibility**; style-only edits → omit code verification). Present as a short checklist for the user — do not complete these items yourself.

## Gotchas

- **Don't rewrite the whole page.** Insert or update the relevant section only. Agents often restructure an entire page when adding a paragraph — preserve existing headings, ordering, and sibling content.
- **PR descriptions overstate scope.** Treat PR body text as a starting hypothesis, not a source of truth. Verify every claim in code before documenting it (Step 5).
- **Config field names drift between PR and code.** PR descriptions sometimes use display names or shorthand; the actual YAML/JSON key in the config struct may differ. Always use the key from code.
- **"Docs exist" ≠ "docs are complete."** A page mentioning the feature doesn't mean it covers the new behavior. Check what the PR actually changed before classifying as `Docs present`.

## Return format

Use this structure:

```markdown
## Files changed
- docs/sources/.../feature.md

## PR-to-doc mapping
| PR | What was documented | File |
|----|---------------------|------|
| #1234 | New config section for `timeout` | `docs/sources/.../feature.md` |

## Open items
- [ ] Verify default value for `timeout` — PR says 30s, code unclear
- [ ] UI layout change needs visual confirmation (defer to screenshot-check)
```

## Reference

- Triage skill: [`../docs-pr-check/SKILL.md`](../docs-pr-check/SKILL.md)
- Step 5 (validate claims): [`references/validate-claims.md`](references/validate-claims.md)
- Repo orientation: `../shared/docs-context-guide.md`
- Workflow detail: `../shared/release-notes-workflow.md`
- Verification checklist: `../shared/verification-checklist.md`
