# Tempo documentation context guide

Read this before any Tempo doc task — writing, updating, or reviewing. It tells you where things are and how to verify what you write. It does **not** cover style (the style guide does) or PR workflow (the PR skills do).

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
│   ├── manifest.md                 # Auto-generated — do not hand-edit (see AGENTS.md)
│   ├── AGENTS.md                   # Agent notes for this directory
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

Use these checks for any doc task — writing, updating, or reviewing.

| What you're verifying | Where to check |
|---|---|
| Config default value | `RegisterFlagsAndApplyDefaults` in the component's `config.go` |
| Config option exists | `docs/sources/tempo/configuration/_index.md` on the current branch |
| Feature introduction version | `CHANGELOG.md` |
| TraceQL syntax is valid | `pkg/traceql/test_examples.yaml` |
| Block format default | `docs/sources/tempo/configuration/parquet.md` |

Always validate claims against code. Do not rely solely on PR descriptions or user-provided text.

## Conventions

Only what is not already in the style guide:

- **Component names**: hyphenated in prose ("live-store", "block-builder", "metrics-generator"). For YAML config keys, verify the exact spelling in the component's `config.go` rather than assuming a convention.
- **Block format naming**: lowercase-v, no space ("vParquet4", "vParquet5")
- **Tone matching**: read 2-3 sibling docs in the same directory before writing to match the existing style and depth

### Required reading before writing or reviewing

- `.agents/doc-agents/shared/style-guide.md` — always
- `modules/generator/AGENTS.md` — when working on metrics-generator docs

## Gotchas

These are non-obvious facts that will cause errors if you assume the obvious:

1. **`manifest.md` is auto-generated.** Refer to `docs/sources/tempo/configuration/AGENTS.md` for details. Only edit `_index.md` for config reference changes.
2. **`modules/ingester/` is legacy.** The ingester is replaced in 3.0 by live-store and block-builder. The code directory still exists pending cleanup — do not reference or document it. Use `architecture.md` for current components.
3. **API parameters keep old names.** Some query parameters (e.g., `mode=ingesters`) retain 2.x names while routing to new components. Check the code (e.g., `modules/frontend/` query handlers) for the current mapping — code is the source of truth.
4. **Block format default vs. latest may differ.** The latest format version in code may not be the default. Always check `docs/sources/tempo/configuration/parquet.md` for the current default.
