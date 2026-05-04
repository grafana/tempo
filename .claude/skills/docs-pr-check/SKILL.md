---
name: docs-pr-check
description: Assess documentation status for PRs — classify gaps, identify existing docs that need updating, and flag UI changes that may affect screenshots. Use when the user asks to check docs coverage for PRs, triage documentation needs, audit PRs for missing docs, assess whether a set of changes needs documentation, or check what recently merged PRs need docs.
metadata:
  use_case: write-or-update
  workflow: evaluate
---

# PR documentation assessment

For a list of PRs, determine whether each has adequate documentation, classify gaps, and identify what needs writing or updating.

## Before you begin

1. Load local context per [`../shared/load-context.md`](../shared/load-context.md).
2. Get PR numbers to check. The user may provide them directly, reference a curated release list or CHANGELOG, or ask for recently merged PRs by time range. If the user gives a time range (for example, "last week," "since Monday"), query for merged PRs:

```bash
gh pr list --repo YOUR_ORG/YOUR_REPO --state merged --search "merged:>YYYY-MM-DD" --json number,title,labels,files --limit 100
```

Filter out bot authors (dependabot, github-actions) and present the list for confirmation before classifying.

## Steps

### 1. Classify each PR

Look up the PR:

```bash
gh pr view XXXX --repo YOUR_ORG/YOUR_REPO --json title,body,files,labels
```

Classify as **needs docs** if the PR introduces: a new user-facing feature, configuration option or flag, changed behavior, API endpoint or query syntax, breaking change or migration step, or new/renamed/repositioned UI element.

Classify as **no docs required** if the PR is: an internal refactor, test-only change, dependency bump, CI/CD change, or performance optimization with no user-visible change.

When the PR metadata doesn't clearly indicate a user-facing change, inspect the diff before classifying as no docs required:

```bash
gh pr diff XXXX --repo YOUR_ORG/YOUR_REPO
```

Look for changes in frontend files (`.tsx`, `.ts`, `.jsx`, `.js`) that modify user-visible elements: button labels, menu items, tab names, form controls, placeholder or tooltip text, accessible names, or new/removed pages and dialogs. If present, reclassify as **needs docs**.

### 2. Check existing documentation coverage

For PRs that need docs:

1. Are there files changed under `docs/`?
2. **Relevant docs:** Under `<your-docs-root>`, is there a page or section for *this* feature or change area (not unrelated docs only)?
3. **Completeness:** If yes, do those pages cover the PR's user-facing delta — behavior, config, examples, and concept/task/reference as your site expects for that scope?

### 3. Assign a category

| Category | Meaning |
|----------|---------|
| Docs present | PR includes docs or existing docs fully cover the feature |
| Docs needed | User-facing change with no documentation |
| Docs update needed | Existing docs are incomplete or don't reflect the new behavior |
| No docs required | Internal change with no user-facing impact |

If relevant docs exist but still have gaps, classify as **Docs update needed**, not **Docs present**.

### 4. Identify specific gaps

For `Docs needed` and `Docs update needed` PRs, search `<your-docs-root>` for the feature name and identify: whether the feature is mentioned but not explained, missing configuration examples, whether an existing page needs updating vs. entirely new content, and which specific files need work.

For PRs with UI changes, also search affected doc pages for image references (`![` or `figure` tags). Record the doc page path, screenshot count, and image filenames — this is the handoff artifact for `screenshot-check`. Skip this if affected pages have no screenshots.

## Return format

**Classification table:**

| PR | Title | Classification | Notes |
|----|-------|---------------|-------|
| #XXXX | ... | Docs present | Link: `<your-docs-root>/...` |
| #XXXX | ... | Docs needed | No docs found; new content needed |
| #XXXX | ... | Docs update needed | `configuration/_index.md` missing new flag |
| #XXXX | ... | No docs required | Internal refactor |

**Gap summary** — prioritized by user impact:
1. PRs where docs are entirely missing
2. Existing pages that need updates
3. PRs where completeness is uncertain and needs engineering input

**Screenshot inventory** (only when UI changes affect pages with screenshots):

| PR | Affected doc page | Screenshot count | Image references |
|----|-------------------|------------------|------------------|
| #XXXX | `<path>` | N | `screenshot-...` |

If screenshots are flagged, offer to run `screenshot-check` on those pages.

## Reference

- Downstream skill: [`../docs-pr-write/SKILL.md`](../docs-pr-write/SKILL.md)
- Repo orientation: `../shared/docs-context-guide.md`
- Workflow detail: `../shared/release-notes-workflow.md` (Phases 1.5–1.75)
- Screenshot validation: `screenshot-check` (if available)
