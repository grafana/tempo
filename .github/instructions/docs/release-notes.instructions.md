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

## Workflow

Follow the full multi-phase workflow defined in [`.claude/skills/shared/release-notes-workflow.md`](../../../.claude/skills/shared/release-notes-workflow.md).

That file is the source of truth for all release notes phases, including CHANGELOG curation (Phase 0), documentation assessment (Phase 1.5), documentation gap resolution (Phase 1.75), writing, validation, patch release handling, example prompts, and the iteration checklist.

## Style

Follow the style conventions in [`.claude/skills/shared/release-notes-workflow.md`](../../../.claude/skills/shared/release-notes-workflow.md) (the "Style guidelines" section) for Tempo-specific naming, PR link format, documentation links, and TraceQL and configuration examples. For general documentation style, refer to [`.claude/skills/shared/style-guide.md`](../../../.claude/skills/shared/style-guide.md).