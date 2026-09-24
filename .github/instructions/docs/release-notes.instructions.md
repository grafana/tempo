---
# Applies to all Tempo release notes pages, including nested version-1/
# and version-2/ directories (for example v3-0.md and version-2/v2-10.md).
applyTo: "**/release-notes/**/*.md"
---

# Tempo release notes workflow

Instructions for creating release notes for Grafana Tempo releases.

## Role

Act as an experienced technical writer and distributed tracing expert for Grafana Labs.

Write release notes for software developers, SREs, and platform engineers who use Tempo for distributed tracing and observability.

Focus on user impact, practical examples, and clear upgrade guidance.

## Style

Use "Grafana Tempo" on first mention, then "Tempo." Use "TraceQL" (not "traceql"), "vParquet4"/"vParquet5" (lowercase v, no space), "metrics-generator" (hyphenated), and reference versions as "Tempo X.Y." Use "refer to," not "see," for links. Use the version placeholder for internal doc links: `/docs/tempo/<TEMPO_VERSION>/path/to/doc/`. Always include PR links: `[[PR 5982](https://github.com/grafana/tempo/pull/5982)]`, or `(PRs [#5939](...), [#6001](...))` for multiple. For general documentation style, refer to [`.claude/skills/shared/style-guide.md`](../../../.claude/skills/shared/style-guide.md).

This style block applies to every edit under `release-notes/`, including small fixes to already-shipped files. It does not require reading the full workflow.

## Workflow

Load the full multi-phase workflow at [`.claude/skills/shared/release-notes-workflow.md`](../../../.claude/skills/shared/release-notes-workflow.md) only when **creating a new version's release notes** or **applying a patch-release update** (for example, adding a `X.Y.Z` security or bug-fix section to an existing file). That file is the source of truth for all release notes phases: source curation (Phase 0, with its own human-review gate), documentation assessment (Phase 1.5), documentation gap resolution (Phase 1.75), writing, validation, patch release handling, example prompts, and the iteration checklist.

Do not load it for style-only edits, typo fixes, or link corrections — the style block above already covers those.