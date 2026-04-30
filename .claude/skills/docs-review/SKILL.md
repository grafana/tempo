---
name: docs-review
description: Review documentation changes for style, accuracy, and completeness. Use when docs have been written or updated and need a quality check before submission, when the user asks to review docs, proofread, check style compliance, or verify technical accuracy of documentation.
metadata:
  use_case: quality-review
  workflow: evaluate
---

# Documentation review

Review docs created or updated by `docs-pr-write` (or any other process) before submission. Provide file paths to review and any open items or uncertain claims from the writing step.

## Before you begin

1. Load local context per [`../shared/load-context.md`](../shared/load-context.md). If no local context file exists and you need to distinguish generated pages from hand-authored ones, ask the user before editing.
2. Read `../shared/style-guide.md`. Apply these rules throughout.

## Steps

### 1. Triage the change

Classify the content being reviewed:

- **(a) Style/editorial only** — wording, formatting, restructuring with no new technical claims. Proceed to step 2.
- **(b) Technical content** — documents config options, default values, syntax, API behavior, or new features. Read `references/technical-verification.md` and follow those steps alongside the style review below.

When in doubt, classify as **(b)**. The overhead of a targeted code check is small compared to publishing an inaccuracy.

### 2. Run Vale

[Vale](https://vale.sh) is a command-line linter for prose. Use your organization's Vale config and rules (path from local context or project root).

If Vale is installed, run it against the changed files:

```bash
vale <file_path>
```

If Vale is not installed, manually check style guide compliance in step 4.

### 3. Check frontmatter

For each changed file, verify:

- `title` and `description` are present
- `aliases` are correct (especially if the file moved)
- Fields match the topicType templates in the style guide

### 4. Check style

Read each changed file. Check against the style guide rules:

- Present tense, active voice, second person
- Sentence case headings
- "refer to" not "see" for links
- Internal links end with "/"
- Admonitions used sparingly

### 5. Check links

Verify internal links resolve to existing pages. Check that cross-links to related pages exist where useful.

### 6. Fix and re-check

Fix any issues found. Re-run Vale (if available) until it passes clean. Re-read changed sections to confirm fixes didn't introduce new problems.

### 7. Present results

Return:

1. Triage classification per file (category a or b from step 1)
2. Issues found, grouped by file (style issues and technical accuracy issues separately)
3. Frontmatter and link check results per file (even if no issues found)
4. Summary of what was changed and why
5. For technical reviews: list of claims verified against code, with any divergences

After review, ask: _"Would you like me to create a PR, or would you prefer to review the changes locally first?"_

## Gotchas

- Vale directives in existing docs are often intentional suppressions. Don't remove them without cause.
- **Generated reference pages** (path from local context): do not hand-edit unless local context says otherwise.
- Release notes may use Vale suppressions at the top of the file by convention.
- The style guide says "refer to" not "see" — but link text in Next Steps sections can use the link text directly without "refer to."

## Reference

- Style guide: `../shared/style-guide.md`
- Technical verification (loaded in step 1 for technical content): `references/technical-verification.md`
- Full verification checklist (human handoff): `../shared/verification-checklist.md`
- Repo orientation: `../shared/docs-context-guide.md`
