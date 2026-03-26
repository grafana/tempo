---
name: docs-review
description: Review documentation changes for style, accuracy, and completeness. Use when docs have been written or updated and need a quality check before submission.
allowed-tools: Bash Read Grep
---

# Documentation review

Review docs created or updated by `docs-pr-write` (or any other process) before submission.

## Usage

Invoke with `/docs-review`.

Provide:
- File paths to review
- Any open items or uncertain claims from the writing step

## Steps

### 1. Read the style guide

Read `.agents/doc-agents/shared/style-guide.md`. Apply these rules throughout.

### 2. Triage the change

Classify the content being reviewed:

- **(a) Style/editorial only** — wording, formatting, restructuring with no new technical claims. Proceed to step 3.
- **(b) Technical content** — documents config options, default values, syntax, API behavior, or new features. Read `references/technical-verification.md` and follow those steps alongside the style review below.

When in doubt, classify as **(b)**. The overhead of a targeted code check is small compared to publishing an inaccuracy.

### 3. Run Vale

[Vale](https://vale.sh) is a command-line linter for prose. Install it from https://vale.sh/docs/install/. The Grafana Vale config and custom rules live in the writers-toolkit repo:
- Config: https://github.com/grafana/writers-toolkit/blob/main/.vale.ini
- Grafana rules: https://github.com/grafana/writers-toolkit/tree/main/vale/Grafana

If Vale is installed, run it against the changed files:

```bash
vale <file_path>
```

If Vale is not installed, manually check style guide compliance in step 5.

### 4. Check frontmatter

For each changed file, verify:
- `title` and `description` are present
- `aliases` are correct (especially if the file moved)
- Fields match the topicType templates in the style guide

### 5. Check style

Read each changed file. Check against the style guide rules:
- Present tense, active voice, second person
- Sentence case headings
- "refer to" not "see" for links
- Internal links end with "/"
- Admonitions used sparingly

### 6. Check links

Verify internal links resolve to existing pages. Check that cross-links to related pages exist where useful.

### 7. Fix and re-check

Fix any issues found. Re-run Vale (if available) until it passes clean. Re-read changed sections to confirm fixes didn't introduce new problems.

### 8. Present results

Return:
1. Triage classification per file (category a or b from step 2)
2. Issues found, grouped by file (style issues and technical accuracy issues separately)
3. Frontmatter and link check results per file (even if no issues found)
4. Summary of what was changed and why
5. For technical reviews: list of claims verified against code, with any divergences

After review, ask: _"Would you like me to create a PR, or would you prefer to review the changes locally first?"_

## Gotchas

- Vale directives in existing docs (`<!-- vale Grafana.We = NO -->`) are intentional suppressions. Don't remove them.
- `manifest.md` is auto-generated. Don't review or edit it.
- Release notes use Vale suppressions at the top of the file by convention.
- The style guide says "refer to" not "see" — but link text in Next Steps sections can use the link text directly without "refer to."

## Reference

- Style guide: `.agents/doc-agents/shared/style-guide.md`
- Technical verification (loaded in step 2 for technical content): `references/technical-verification.md`
- Full verification checklist (human handoff): `.agents/doc-agents/shared/verification-checklist.md`
- Repo orientation: `.agents/doc-agents/shared/docs-context-guide.md`
- Vale config source: https://github.com/grafana/writers-toolkit
