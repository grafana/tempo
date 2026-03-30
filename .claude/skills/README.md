# Skills: PR docs workflow

This repository uses a three-step pipeline for PR-driven documentation work:

1. `docs-pr-check`: triage and classify documentation status per PR
2. `docs-pr-write`: write or update documentation for the PRs that need work
3. `docs-review`: review the changes for style, accuracy, and completeness

Use `/docs-workflow` to run all three steps in sequence, or invoke each skill individually.

## Quick start

```text
/docs-workflow
```

Provide a list of PR numbers and a target branch. The workflow runs check → write → review and asks for your input between each step. See `docs-workflow/SKILL.md` for details.

## Individual skills

### 1) Triage — `/docs-pr-check`

Input:
- Curated PR list for the release train

Output:
- Classification table (`Docs present`, `Docs needed`, `Docs update needed`, `No docs required`)
- Prioritized gap summary

### 2) Write — `/docs-pr-write`

Input:
- Only PRs marked `Docs needed` or `Docs update needed` from step 1
- Branch/version context (for example `main`, `release-2.10`, `3.0-docs`)

Output:
- Updated docs files
- PR-to-doc mapping
- Open issues/blockers needing engineering clarification

### 3) Review — `/docs-review`

Input:
- File paths changed in step 2
- Any open items or uncertain claims from the writing step

Output:
- Issues found, grouped by file
- Summary of changes

## Why split into three skills

Keeping triage, writing, and review separate improves quality and repeatability:

- `docs-pr-check` stays fast and objective (classification + gap detection).
- `docs-pr-write` stays focused on content creation (target page, examples, validation).
- `docs-review` applies a consistent quality bar (style, frontmatter, links, accuracy).
- The handoffs create a clear audit trail of what was evaluated, what was changed, and what was reviewed.

## Handoff contract

| From → To | What passes |
|-----------|-------------|
| Check → Write | PR number, classification, gap note, suggested target files |
| Write → Review | Changed file paths, open items or uncertain claims |

Example handoff row (check → write):

```text
#5962 | Docs update needed | API docs missing new Accept header behavior | docs/sources/tempo/api_docs/_index.md
```

## Scope boundaries

- `docs-pr-check` determines if docs work is needed and where gaps exist.
- `docs-pr-write` implements the required docs updates and validates against code.
- `docs-review` checks the result for style, frontmatter, links, and accuracy.

These skills and the `/docs-workflow` command shouldn't be used as a substitute for release-notes assembly unless explicitly requested.

## When to use this vs. the writer-agent

| Task | Use |
|------|-----|
| PRs shipped and need docs | `/docs-workflow` or individual skills |
| New feature or product docs from scratch | Writer-agent (`.agents/doc-agents/writers/writer-agent.md`) |
| General documentation work not tied to PRs | Writer-agent |

## Evals

Each skill has an `evals/evals.json` with test cases for evaluating output quality. See `evals/README.md` for how to select test inputs, run evals, and grade results.

## Repo orientation

Before starting any doc task, read `.agents/doc-agents/shared/docs-context-guide.md` for code-to-docs mapping, key file paths, verification patterns, and Tempo conventions.

## Tips

- Start with a manually curated PR list so non-user-facing PRs are filtered out early.
- Prefer updating existing docs pages over creating new ones.
- Validate defaults, `configuration` keys, and endpoint details against code before finalizing docs.
