# Integration Evaluation: docs-pr-check → docs-pr-write (Tempo 3.0)

**PR list:** #6103, #5962, #5840  
**Target branch:** main  
**Session branch:** evals/docs-pr-write-iteration-1

---

## Phase 1.5 — Documentation Assessment (docs-pr-check)

### PR Lookup Results

**PR #6103 — "Initial support for analyzing and configuring integer dedicated columns"**
- Introduces new `tempo-cli analyse block` functionality for integer dedicated columns
- New CLI flags: `--num-int-attr`, `--include-well-known`, `--int-percent-threshold`, `--str-percent-threshold`, `--generate-cli-args`
- Adds 5% prevalence heuristic for integer column recommendations
- Documentation checkbox: **NOT checked** in PR body

**PR #5962 — "Accept: application/llm"**
- Adds `application/llm` Accept header support to 2 API endpoints (Query V1, Query V2)
- MCP server automatically sets `application/llm` header in its tool calls
- Documentation checkbox: **NOT checked** in PR body

**PR #5840 — "build(deps): bump cloud.google.com/go/storage from 1.56.0 to 1.57.1"**
- Dependabot dependency bump; no user-facing changes
- Labels: `dependencies`, `go`

### Existing Docs Check

**PR #6103:**
- `docs/sources/tempo/operations/tempo_cli.md` (lines 606–618): Already documents `--num-int-attr`, `--int-percent-threshold`, `--include-well-known`, `--generate-cli-args`, `--str-percent-threshold` flags for `analyse block` and `analyse blocks`
- `docs/sources/tempo/operations/dedicated_columns.md` (lines 72–123): Has a full "Integer attribute selection" section explaining the 5% threshold heuristic, example output with ✅ recommendation markers, and example configuration YAML

**PR #5962:**
- `docs/sources/tempo/api_docs/_index.md` (lines 171–174, 220–223, 625–627): Already documents `Accept: application/vnd.grafana.llm` for Query V1, Query V2, and tag values endpoints
- `docs/sources/tempo/api_docs/mcp-server.md`: Does **not** mention that MCP tools automatically request LLM-formatted responses when querying trace data

### Classification Table

| PR | Title | Classification | Notes |
|----|-------|----------------|-------|
| #6103 | Initial support for analyzing and configuring integer dedicated columns | **Docs present** | CLI flags documented in `tempo_cli.md`; 5% integer heuristic and examples in `dedicated_columns.md` |
| #5962 | Accept: application/llm | **Docs update needed** | Query v1/v2 Accept headers already in `api_docs/_index.md`; gap: `mcp-server.md` does not document that MCP auto-sets `application/vnd.grafana.llm` on trace queries |
| #5840 | build(deps): bump cloud.google.com/go/storage | **No docs required** | Dependency bump with no user-facing change |

### Gap Summary

**Priority 1 — Docs update needed:**
1. **PR #5962**: `docs/sources/tempo/api_docs/mcp-server.md` needs a section explaining that the MCP server automatically requests LLM-optimized trace responses using the `Accept: application/vnd.grafana.llm` header. Users need to understand: (a) MCP handles this automatically without manual configuration, (b) the format is compact/LLM-readable vs full OTel JSON, (c) it should not be used for programmatic integrations, (d) they can request the same format directly from Query V1/V2 endpoints.

**No further action needed:**
- #6103: Docs are comprehensive in both `tempo_cli.md` and `dedicated_columns.md`
- #5840: Dependency bump; no user-visible change

---

## Phase 1.75 — Documentation Execution (docs-pr-write)

**PRs passed from Phase 1.5:** #5962 only

### Scope and Order

Only one PR requires documentation work. PR #5962 is classified as "Docs update needed" — updating an existing page (`mcp-server.md`) rather than creating new content.

### PR #5962 — Capability Reconstruction

**What users can do now (from PR body + file changes):**
- Tempo MCP server tools (`mcp_tools.go`) automatically set `Accept: application/llm` on trace fetch requests
- This triggers a compact, LLM-friendly JSON response instead of full OpenTelemetry JSON
- Two primary endpoints affected: `GET /api/traces/<traceID>` (Query V1) and `GET /api/v2/traces/<traceID>` (Query V2)
- The format is explicitly unstable — "subject to change" and "not to be relied on" per PR description
- Direct API users can also opt into this format manually via the Accept header

**Note on MIME type:** The PR body uses `application/llm`; existing API docs and the docs update use `application/vnd.grafana.llm` (the properly namespaced MIME type). This may reflect the actual registered value in `pkg/api/http.go`. The docs use the more correct form.

### Target File

**Existing page updated:** `docs/sources/tempo/api_docs/mcp-server.md`

Rationale: This is the canonical home for MCP server documentation. Users enabling the MCP server will read this page; it's the right place to explain what format MCP uses when fetching trace data.

### Change Made

Added a new `## LLM-optimized response format` section at the end of `mcp-server.md`:

```markdown
## LLM-optimized response format

When the MCP server fetches trace data, it automatically requests an LLM-optimized response format by sending `Accept: application/vnd.grafana.llm` to the Tempo query endpoints. This produces a compact JSON representation of the trace that is easier for LLMs to process than the full OpenTelemetry JSON format.

This format is intended for LLM consumption only. It is subject to change and should not be used for programmatic integrations.

You can also request this format directly from the [Query V1 and Query V2 endpoints](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/#query) by setting the `Accept: application/vnd.grafana.llm` header in your request.
```

### Files Changed

| File | Change |
|------|--------|
| `docs/sources/tempo/api_docs/mcp-server.md` | Added `## LLM-optimized response format` section explaining automatic LLM format in MCP tools, format stability warning, and cross-link to API docs |

### PR-to-Doc Mapping

| PR | What was documented | Where |
|----|--------------------|----|
| #5962 | MCP server auto-sets `application/vnd.grafana.llm` Accept header; format is LLM-only and subject to change; direct API usage also supported | `docs/sources/tempo/api_docs/mcp-server.md` |

### Open Items

1. **MIME type verification**: The PR body says `application/llm`; the docs use `application/vnd.grafana.llm`. This should be confirmed against the actual string constant in `pkg/api/http.go` in grafana/tempo. If the code uses `application/llm`, the API docs and this page need updating.
2. **Tag values endpoint**: The existing API docs also show `application/vnd.grafana.llm` for the tag values endpoint (line 627 of `api_docs/_index.md`). It's unclear whether PR #5962 covers tag values or if that was added in a separate PR. Engineering confirmation would clarify scope.

---

## Handoff Behavior Assessment

**Did Phase 1.5 → Phase 1.75 happen automatically without user re-entry?**

Yes. The workflow proceeded in a single session:
1. Phase 1.5 classified all three PRs and identified #5962 as the only PR requiring documentation work
2. Phase 1.75 began immediately using the classification output from Phase 1.5
3. No user prompt or re-entry was required between the two phases
4. Only PRs classified as "Docs needed" or "Docs update needed" were passed to Phase 1.75 (#5840 and #6103 were correctly excluded)
