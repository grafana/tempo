---
name: docs-pr-check
description: Assess documentation status for each PR in a Tempo release — classify gaps and identify existing docs that need updating
allowed-tools: Bash Read Grep
---

# Release Notes: Documentation Assessment (Phases 1.5–1.75)

For a given PR list, determine whether each PR has adequate documentation, classify gaps, and identify specific updates needed in existing docs.

## Usage

Invoke with `/docs-pr-check`

Provide a list of PR numbers (from the curated list produced by `/rn-curate`). Ask the user for the PR list if not provided.

## Steps to Perform

### 1. For each PR, determine if documentation is needed

Look up the PR:

```bash
gh pr view XXXX --repo grafana/tempo --json title,body,files,labels
```

Classify as **needs docs** if the PR introduces:
- A new user-facing feature
- A new configuration option or flag
- Changed behavior
- A new API endpoint or query syntax
- A breaking change or migration step

Classify as **no docs required** if the PR is:
- An internal refactor
- A test-only change
- A dependency bump
- A CI/CD change
- A performance optimization with no user-visible change

### 2. For PRs that need docs, check if documentation exists

1. **Check the PR checklist**: Is "Documentation added" checked in the PR body?
2. **Check PR file changes**: Are there any files changed under `docs/`?
3. **Search existing docs** for the feature name in `docs/sources/tempo`.
4. **Assess completeness**: If docs exist, do they cover the new behavior, configuration options, and examples?

### 3. Classify each PR into one of four categories

| Category | Meaning | Action |
|----------|---------|--------|
| Docs present | PR includes docs or existing docs fully cover the feature | Link to docs in release notes |
| Docs needed | User-facing change with no documentation | Flag for documentation work |
| Docs update needed | Existing docs are incomplete or don't reflect the new behavior | Update existing docs as part of the release |
| No docs required | Internal change with no user-facing impact | No action needed |

If docs exist but you identify any actionable gap, classify as **Docs update needed**, not **Docs present**.

### 4. For "docs needed" and "docs update needed" PRs, identify specific gaps

Search `docs/sources/tempo` for the feature name to check if it's partially documented. Identify:

- Feature mentioned but not explained?
- Missing configuration examples?
- Existing page that needs updating vs. entirely new content needed?
- Which specific file(s) need to be created or updated?

### 5. Return results

Return two things:

**Table:**

| PR | Title | Classification | Notes |
|----|-------|---------------|-------|
| #XXXX | ... | Docs present | Link: `/docs/tempo/<TEMPO_VERSION>/...` |
| #XXXX | ... | Docs needed | No docs found; new content needed |
| #XXXX | ... | Docs update needed | `configuration/_index.md` missing new flag |
| #XXXX | ... | No docs required | Internal refactor |

**Gap summary:**

A prioritized list of documentation work needed, ordered by user impact:
1. PRs where docs are entirely missing (highest priority)
2. Existing pages that need updates
3. PRs where doc completeness is uncertain and needs engineering clarification

## Reference

- Workflow detail: `.agents/doc-agents/shared/release-notes-workflow.md` (Phases 1.5–1.75)
- Repo orientation: `.agents/doc-agents/shared/docs-context-guide.md` — code-to-docs mapping, key file paths, and Tempo doc conventions
