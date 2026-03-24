# Skills: PR docs workflow

This repository uses two complementary skills for PR-driven documentation work:

- `docs-pr-check`: triage and classify documentation status per PR
- `docs-pr-write`: write or update documentation for the PRs that need work

Use them together as a two-stage workflow.

## Why split into two skills

Keeping triage and writing separate improves quality and repeatability:

- `docs-pr-check` stays fast and objective (classification + gap detection).
- `docs-pr-write` stays focused on content creation (target page, examples, validation).
- The handoff creates a clear audit trail of what was evaluated vs what was changed.

## Recommended flow

### 1) Run triage

Invoke:

```text
/docs-pr-check
```

Input:
- Curated PR list for the release train

Output:
- Classification table (`Docs present`, `Docs needed`, `Docs update needed`, `No docs required`)
- Prioritized gap summary

### 2) Run writing execution

Invoke:

```text
/docs-pr-write
```

Input:
- Only PRs marked `Docs needed` or `Docs update needed` from step 1
- Branch/version context (for example `main`, `release-2.10`, `3.0-docs`)

Output:
- Updated docs files
- PR-to-doc mapping
- Open issues/blockers needing engineering clarification

### 3) Run final docs review

Before finalizing, run the Grafana AI Kit `doc-review` skill for quality checks (style, clarity, links, and consistency):

- [doc-review skill](file:///Users/kim-nylander/.claude/plugins/cache/grafana-ai-kit/grafana-tech-writing/1.0.0/skills/doc-review/SKILL.md)

## Handoff contract (from check to write)

Pass this per PR:

- PR number
- Classification
- Gap note (what is missing/incomplete)
- Suggested target docs files, if known

Example handoff row:

```text
#5962 | Docs update needed | API docs missing new Accept header behavior | docs/sources/tempo/api_docs/_index.md
```

## Scope boundaries

`docs-pr-check` should:
- determine if docs work is needed
- identify where docs are missing/incomplete
- prioritize docs work

`docs-pr-write` should:
- implement the required docs updates
- validate docs claims against code
- add concise examples and cross-links

Neither skill should be used as a substitute for release-notes assembly unless explicitly requested.

## Tips

- Start with a manually curated PR list so non-user-facing PRs are filtered out early.
- Prefer updating existing docs pages over creating new ones.
- Validate defaults, `configuration` keys, and endpoint details against code before finalizing docs.
