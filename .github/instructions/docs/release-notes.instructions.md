---
applyTo: "**/release-notes/*.md"
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

## Style quick reference

### Tempo-specific conventions

- Use "Grafana Tempo" on first mention, then "Tempo"
- Use "TraceQL" (not "traceql" or "Trace QL")
- Use "vParquet4", "vParquet5" (lowercase v, no space)
- Use "metrics-generator" (hyphenated)
- Reference versions as "Tempo 2.10" or "Tempo X.Y"

### PR link format

Always include PR links at the end of entries:

```markdown
[[PR 5982](https://github.com/grafana/tempo/pull/5982)]
```

For multiple PRs:

```markdown
(PRs [#5939](https://github.com/grafana/tempo/pull/5939), [#6001](https://github.com/grafana/tempo/pull/6001))
```

### Documentation links

Use the version placeholder for internal docs:

```markdown
[documentation](/docs/tempo/<TEMPO_VERSION>/path/to/doc/)
```

### TraceQL examples

Format TraceQL queries in code blocks with the `traceql` language tag:

````markdown
```traceql
{ span:childCount > 10 }
```
````

### Configuration examples

Use YAML code blocks for configuration:

```yaml
storage:
  trace:
    block:
      version: vParquet5
```