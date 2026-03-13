# docs-pr-write Eval — PR #5962

## Files Changed

- `docs/sources/tempo/api_docs/_index.md`

## PR-to-Doc Mapping

| PR | What was documented | Where |
|----|--------------------|----|
| [#5962](https://github.com/grafana/tempo/pull/5962) — `Accept: application/llm` | Added `Accept: application/vnd.grafana.llm` to the **Returns** section of the Query v1 endpoint (`GET /api/traces/<traceid>`). The header returns a simplified JSON format optimized for LLM consumption. Reformatted the returns block to match the Query V2 section style (bulleted list of Accept values). | `docs/sources/tempo/api_docs/_index.md`, Query section (formerly lines 167–171) |

## Context: What Was Already Documented

The doc update commit `fd1d94530` (merged Jan 2026) had already added `application/vnd.grafana.llm` to:
- **Query V2** (`GET /api/v2/traces/<traceid>`) — `Accept` header list
- **Search tag values V2** (`GET /api/v2/search/tag/<tag>/values`) — `Accept` header list with a **Returns** block

## What This Eval Fixed

The **Query v1** endpoint (`GET /api/traces/<traceid>`) was the remaining gap. Its **Returns** section only mentioned `application/protobuf` and did not document `application/vnd.grafana.llm`, even though both endpoints use the same `TraceByIDResponse` combiner that supports the LLM format.

The change:
- Rewrote the single-sentence returns description into a structured `Accept` header list (matching the style of Query V2)
- Added `Accept: application/vnd.grafana.llm` with the canonical disclaimer ("subject to change, not for programmatic use")

## Open Items

- **Header value in PR title vs. code**: The PR title and body use `application/llm`, but the actual header constant in code is `application/vnd.grafana.llm`. Docs use the correct code value.
- **MCP server auto-sets header**: The PR also updated `modules/frontend/mcp_tools.go` to automatically set `Accept: application/vnd.grafana.llm`. This is an implementation detail for the MCP server, already linked from the API table (`/api/mcp`). No additional doc change needed unless the MCP server page (`api_docs/mcp-server`) should mention the automatic header behavior — deferred.
- **v1 endpoint LLM format parity note**: The v1 trace endpoint and v2 trace endpoint both support `application/vnd.grafana.llm`, but the v1 returns a `TraceByIDResponse` wrapper with metrics, whereas v2 returns the same structure. No user-visible difference in LLM output was found in the code; no additional note needed.
