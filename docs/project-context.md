<!--  This file is used to provide context to the documentation agent. -->

# Project context (local)

## Identity

- **Product name (short):** Tempo
- **Product name (first mention in prose):** Grafana Tempo
- **GitHub org/repo:** grafana/tempo (upstream), knylander-grafana/tempo-doc-work (fork)

## Branches and releases

- **Default development branch:** `main`
- **Release branch pattern:** `release-vX.Y` (for example, `release-v2.10`)
- **Docs version mapping:** each `release-vX.Y` branch maps to the `vX.Y.x` version on grafana.com/docs/tempo/

## Documentation paths

- **Documentation root (filesystem):** `docs/sources/tempo/`
- **Helm chart docs:** `docs/sources/helm-charts/tempo-distributed/`
- **Shared partials (headless):** `docs/sources/tempo/shared/` (reusable snippets included by other pages)
- **Generated pages (do not hand-edit):** `docs/sources/tempo/configuration/manifest.md` — refer to `docs/sources/tempo/configuration/AGENTS.md` for details. Only edit `_index.md` for config reference changes.
- **Configuration reference:** `docs/sources/tempo/configuration/`
- **Changelog:** `CHANGELOG.md` at repo root
- **Architecture / "start here" page:** `docs/sources/tempo/reference-tempo-architecture/`
- **Design proposals (internal, not published):** `docs/design-proposals/`
- **Doc build tooling:** `docs/Makefile`, `docs/docs.mk`, `docs/variables.mk`, `docs/make-docs`

## Documentation tree

The published docs live under `docs/sources/tempo/`. Major sections:

| Section path | Title | Covers |
|-------------|-------|--------|
| `introduction/` | Introduction | Architecture, glossary, telemetry, Tempo in Grafana, trace structure |
| `solutions-with-traces/` | Use traces to find solutions | Use-case pages: metrics queries, app insights, diagnose errors |
| `set-up-for-tracing/` | Set up for tracing | Instrument → collectors → deploy Tempo → validate |
| `set-up-for-tracing/instrument-send/` | Instrument for distributed tracing | Choose method, set up instrumentation, set up collector (Alloy, OTel, sampling), test |
| `set-up-for-tracing/setup-tempo/` | Set up Tempo | Plan, deploy (locally, K8s/Helm/Operator/Tanka), validate, upgrade, CLI flags |
| `traceql/` | TraceQL | Query language: constructing/tuning queries, architecture, query editor in Grafana |
| `metrics-from-traces/` | Metrics from traces | Metrics-generator, span metrics, service graphs, TraceQL metrics |
| `configuration/` | Configure Tempo | Core config, Parquet, tenant IDs, usage, network options, hosted storage (S3, GCS, Azure) |
| `operations/` | Manage Tempo | Auth, caching, backend search, schema, CLI, vulture, dedicated columns, ingestion, monitoring, advanced systems |
| `reference-tempo-architecture/` | Tempo architecture reference | Architecture overview, deployment modes, object storage, block format, partition ring, components |
| `api_docs/` | Tempo HTTP API | HTTP ingest endpoints, MCP server doc |
| `release-notes/` | Release notes | v2-0 through v2-10, plus `version-1/` (v1-2 through v1-5) |
| `troubleshooting/` | Troubleshoot Tempo | Sending traces, querying issues |
| `community/` | Community | Community content |

Helm chart docs under `docs/sources/helm-charts/tempo-distributed/`:

| Section path | Title |
|-------------|-------|
| (root) | Grafana tempo-distributed Helm chart documentation |
| `get-started-helm-charts/` | Get started with Grafana Tempo using the Helm chart |
| `release-notes/` | Grafana Tempo Helm chart release notes |

## Code ↔ documentation mapping

| Code area | Documentation area |
|-----------|---------------------|
| `pkg/traceql/` | `docs/sources/tempo/traceql/` |
| `modules/generator/` | `docs/sources/tempo/metrics-from-traces/metrics-generator/` |
| `modules/distributor/` | `docs/sources/tempo/reference-tempo-architecture/components/distributor/` |
| `modules/querier/` | `docs/sources/tempo/reference-tempo-architecture/components/querier/` |
| `modules/frontend/` | `docs/sources/tempo/reference-tempo-architecture/components/query-frontend/` |
| `modules/ingester/` | `docs/sources/tempo/reference-tempo-architecture/components/` |
| `modules/blockbuilder/` | `docs/sources/tempo/reference-tempo-architecture/components/block-builder/` |
| `modules/livestore/` | `docs/sources/tempo/reference-tempo-architecture/components/live-store/` |
| `modules/cache/` | `docs/sources/tempo/operations/` (caching) |
| `modules/overrides/` | `docs/sources/tempo/operations/manage-advanced-systems/` (user-configurable overrides) |
| `modules/storage/` | `docs/sources/tempo/reference-tempo-architecture/` (object storage, block format) |
| `tempodb/` | `docs/sources/tempo/reference-tempo-architecture/` (block format, compaction) |
| `cmd/tempo-cli/` | `docs/sources/tempo/operations/tempo_cli/` |
| `cmd/tempo-vulture/` | `docs/sources/tempo/operations/` (vulture) |
| `cmd/tempo/` | `docs/sources/tempo/configuration/` |
| `operations/` (repo root) | `docs/sources/helm-charts/tempo-distributed/` (Helm/Jsonnet ops tooling) |

## Code validation paths

Paths the agent should check when validating documentation claims against code.

| What to validate | Where to look |
|-----------------|---------------|
| TraceQL syntax / grammar | `pkg/traceql/lexer.go`, `pkg/traceql/parse.go`, `pkg/traceql/expr.y.go` |
| TraceQL types / intrinsics | `pkg/traceql/ast.go`, `pkg/traceql/enum_attributes.go`, `pkg/traceql/enum_statics.go` |
| TraceQL operators | `pkg/traceql/enum_operators.go` |
| TraceQL aggregates / hints | `pkg/traceql/enum_aggregates.go`, `pkg/traceql/enum_hints.go` |
| TraceQL metrics engine | `pkg/traceql/engine_metrics.go`, `pkg/traceql/engine_metrics_functions.go` |
| Configuration structs / defaults | `modules/*/config.go`, `cmd/tempo/app/*.go` |
| Metrics-generator config | `modules/generator/` |
| API endpoints | `cmd/tempo/`, `modules/frontend/` |
| Distributor / ingester behavior | `modules/distributor/`, `modules/ingester/` |
| Storage / compaction | `tempodb/`, `modules/storage/` |

## Frontmatter and site conventions

- **Frontmatter fields:** `title`, `menuTitle`, `description`, `keywords`, `weight`, `aliases`
- **Weight / ordering:** lower `weight` values appear first in navigation
- **Internal link style:** relative paths ending in `/` (not `.md`)
- **Section pages:** named `_index.md`; leaf pages are named `index.md` or `<slug>.md`
- **Shared partials:** `docs/sources/tempo/shared/` contains headless snippets reused via Hugo `readfile` shortcode
- **Admonitions:** `{{</* admonition type="note|caution|warning" */>}}`
- **Public preview:** `{{</* docs/public-preview product="<PRODUCT>" */>}}`

## Conventions for agents

- **Query language:** TraceQL (always capitalized, one word)
- **Storage format versions:** vParquet2, vParquet3, vParquet4, vParquet5 (lowercase "v", camelCase "Parquet", version digit)
- **Block format:** always "block format" or "Parquet block format" (not "block type")
- **Product references:** "Grafana Tempo" on first mention, "Tempo" thereafter; always "Grafana Cloud" (never just "Cloud"); always "Grafana Alloy" (not "Alloy" alone on first mention)
- **Vale / linter config:** no `.vale.ini` in this repo; lint rules come from the docs build pipeline

## Agent and skill resources

| Location | Purpose |
|----------|---------|
| `.agents/guidance/coding.md` | Go coding standards |
| `.agents/guidance/code-review.md` | Code review standards |
| `.agents/guidance/precommit.md` | Pre-commit checklist |
| `.claude/skills/shared/style-guide.md` | Docs style guide |
| `.claude/skills/shared/release-notes-workflow.md` | Release notes process |
| `.claude/skills/shared/verification-checklist.md` | Docs verification |
| `.claude/skills/shared/best-practices.md` | Docs best practices |
| `.claude/skills/shared/docs-context-guide.md` | Docs context guide |
| `.claude/skills/shared/load-context.md` | Context loading instructions |
| `.claude/skills/shared/personas.md` | Persona and intent model |
| `.claude/skills/shared/agent_personas.yaml` | Structured persona data |
| `.claude/skills/` | Agent skills: docs-pr-check, docs-pr-write, docs-review, docs-workflow, persona-check, fix-vendor-conflicts, update-go-version |
| `.github/instructions/docs/` | DOCS.md, release-notes.instructions.md, toolkit.instructions.md |

## Subsystem knowledge

- **Metrics-generator:** `.agents/doc-agents/shared/` contains metrics-generator knowledge; code in `modules/generator/`
- **TraceQL engine:** `pkg/traceql/` — AST, parser, lexer, metrics engine
- **Storage / tempodb:** `tempodb/` — block format, compaction, encoding; docs in `reference-tempo-architecture/`
- **Components:** `modules/` subdirectories map to `reference-tempo-architecture/components/` doc pages

## Gotchas

Non-obvious facts that cause errors if you assume the obvious:

1. **`modules/ingester/` is legacy.** The ingester is replaced in 3.0 by live-store and block-builder. The code directory still exists pending cleanup — do not reference or document it. Use `reference-tempo-architecture/` for current components.
2. **API parameters keep old names.** Some query parameters (for example, `mode=ingesters`) retain 2.x names while routing to new components. Check the code (for example, `modules/frontend/` query handlers) for the current mapping — code is the source of truth.
3. **Block format default vs. latest may differ.** The latest format version in code may not be the default. Always check `docs/sources/tempo/configuration/parquet.md` for the current default.
4. **Helm chart docs live in three places:**
   - `docs/sources/helm-charts/tempo-distributed/` — primary procedures
   - `docs/sources/tempo/set-up-for-tracing/setup-tempo/deploy/kubernetes/helm-chart/` — landing page that links to the primary procedures
   - `docs/sources/tempo/configuration/` — storage, TLS, and network config that mentions Helm values

## Shared features across doc trees

Features that require coordinated updates across the Tempo docs and Helm chart docs:

- **Deployment configuration** — changes to Tempo config may affect Helm chart values and vice versa
- **Release notes** — both `docs/sources/tempo/release-notes/` and `docs/sources/helm-charts/tempo-distributed/release-notes/` must be updated for major releases
- **Upgrade instructions** — `set-up-for-tracing/setup-tempo/upgrade.md` may reference Helm-specific upgrade steps
