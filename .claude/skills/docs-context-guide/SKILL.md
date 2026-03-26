---
name: docs-context-guide
description: Structural orientation for Tempo documentation — maps code to docs, surfaces key files, and encodes non-obvious conventions. Read this before any Tempo doc task.
allowed-tools: Bash Read Grep
---

# Tempo documentation context guide

Use this skill whenever you work on Tempo documentation — writing, updating, or reviewing. It tells you where things are and how to verify what you write. It does **not** cover style (the style guide does) or PR workflow (the PR skills do).

## Orientation — where to look

### Architecture and component names

Read `docs/sources/tempo/introduction/architecture.md` first. Use whatever component names and data-flow descriptions that page uses. Do not invent terminology.

Key components in Tempo 3.0:

| Code module | Role |
|---|---|
| `modules/distributor/` | Receives spans, writes to Kafka |
| `modules/livestore/` | Serves recent traces (replaces ingester for reads) |
| `modules/blockbuilder/` | Builds Parquet blocks from Kafka (replaces ingester for writes-to-storage) |
| `modules/backendscheduler/` | Assigns compaction/maintenance work |
| `modules/backendworker/` | Executes compaction/maintenance work |
| `modules/querier/` | Queries live-stores and object storage |
| `modules/frontend/` | Query frontend, splits/retries |
| `modules/generator/` | Generates metrics from traces |

### Documentation layout

```
docs/sources/tempo/
├── introduction/architecture.md    # Start here
├── configuration/
│   ├── _index.md                   # Main config reference (manually edited)
│   ├── manifest.md                 # Auto-generated — do not hand-edit
│   └── parquet.md                  # Block format versions and defaults
├── traceql/                        # TraceQL query language docs
├── set-up-for-tracing/             # Deployment and setup guides
├── operations/                     # Operational guides
└── release-notes/                  # Per-version release notes

docs/sources/helm-charts/
└── tempo-distributed/              # Primary Helm chart docs
```

### Code-to-docs patterns

- Components live in `modules/<name>/`, config in that module's `config.go`
- Config options are documented in `docs/sources/tempo/configuration/_index.md`
- TraceQL: code in `pkg/traceql/` → docs in `docs/sources/tempo/traceql/`
- Helm chart docs live in three places:
  1. `docs/sources/helm-charts/tempo-distributed/` — primary procedures
  2. `docs/sources/tempo/set-up-for-tracing/setup-tempo/deploy/kubernetes/helm-chart/` — landing page that links to (1)
  3. `docs/sources/tempo/configuration/` — storage, TLS, and network config that mentions Helm values

## How to verify what you write

| Claim type | Where to check |
|---|---|
| Config default value | `RegisterFlagsAndApplyDefaults` in the component's `config.go` |
| Config option exists | `docs/sources/tempo/configuration/_index.md` on the current branch |
| Feature introduction version | `CHANGELOG.md` |
| TraceQL syntax is valid | `pkg/traceql/test_examples.yaml` |
| Block format default | `docs/sources/tempo/configuration/parquet.md` |

Always validate claims against code. Do not rely solely on PR descriptions or user-provided text.

## Conventions

Only what is not already in the style guide:

- **Component names**: hyphenated in prose ("live-store", "block-builder", "metrics-generator"), underscores in YAML config keys (`live_store`, `block_builder`)
- **Block format naming**: lowercase-v, no space ("vParquet4", "vParquet5")
- **Tone matching**: read 2-3 sibling docs in the same directory before writing to match the existing style and depth

### Required reading before writing or reviewing

- `.agents/doc-agents/shared/style-guide.md` — always
- `.agents/doc-agents/shared/metrics-generator-knowledge.md` — when working on metrics-generator docs

### Related resource

- `.agents/doc-agents/shared/verification-checklist.md` — broad, human-facing review checklist. Used by `docs-pr-write` (Step 8) as a handoff artifact. Do not load or execute as part of this skill's workflow.

## Gotchas

These are non-obvious facts that will cause errors if you assume the obvious:

1. **`manifest.md` is auto-generated.** `docs/sources/tempo/configuration/manifest.md` is generated from code. Only edit `_index.md` for config reference changes.
2. **`modules/ingester/` is legacy.** The ingester is replaced in 3.0 by live-store and block-builder. The code directory still exists pending cleanup — do not reference or document it. Use `architecture.md` for current components.
3. **API parameters keep old names.** Some query parameters (e.g., `mode=ingesters`) retain 2.x names while routing to new components. Check the API docs for the current mapping.
4. **Block format default vs. latest may differ.** The latest format version in code may not be the default. Always check `parquet.md` for the current default.

## Guardrails

All doc skills share these rules:

- After writing or updating files, present the list of changed files and ask the user whether they want to create a PR or review the changes locally first.

## Workflow

1. **Orient**: read `architecture.md` and `configuration/_index.md`
2. **Scope**: identify what changed — diff, user description, or GitHub issue
3. **Find**: search `docs/sources/tempo/` for existing content. Prefer updating in place over creating new pages.
4. **Write**: draft the content
5. **Validate**: verify claims against `config.go`, `CHANGELOG.md`, or `test_examples.yaml` as appropriate
6. **Prompt**: present changed files and ask the user whether to create a PR or review locally first.

## Related skills

- **`docs-pr-check`** (`.claude/skills/docs-pr-check/SKILL.md`) — evaluates whether a PR needs documentation. Usable independently or as part of release workflow.
- **`docs-pr-write`** (`.claude/skills/docs-pr-write/SKILL.md`) — writes or updates documentation for PR changes. Usable independently or as part of release workflow.

This skill provides the structural knowledge that those PR-scoped skills can leverage. Use this skill first for orientation, then invoke a PR skill if the task is PR-driven.
