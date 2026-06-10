# Tempo — Agent Guidance

## Coding Standards

Before writing or modifying Go code, read [`.agents/guidance/coding.md`](.agents/guidance/coding.md).

## Code Review Standards

Before reviewing code, read [`.agents/guidance/code-review.md`](.agents/guidance/code-review.md).

## Pre-Commit Checklist

Before pushing or opening a PR, read [`.agents/guidance/precommit.md`](.agents/guidance/precommit.md).

## Changelog Entries

When adding or updating a `CHANGELOG.md` entry, follow [`.agents/skills/changelog-entry/SKILL.md`](.agents/skills/changelog-entry/SKILL.md).

Non-negotiable summary:
- Add entries under `## main / unreleased` only for user-facing or release-noteworthy changes.
- Do not add changelog entries for docs-only changes.
- Use only `[CHANGE]`, `[FEATURE]`, `[ENHANCEMENT]`, or `[BUGFIX]`.
- Keep groups in this order: `[CHANGE]`, `[FEATURE]`, `[ENHANCEMENT]`, `[BUGFIX]`.
- Do not invent tags such as `[SECURITY]`.
- Use one concise bullet line with PR number, PR link, and GitHub author.
