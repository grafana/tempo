# Phase 1.75 Handoff: Documentation Assessment for Tempo 3.0 PRs

**Phase:** 1.5 → 1.75  
**PRs reviewed:** #6103, #5962, #5840  
**Date:** 2026-03-13

---

## Classification Table

| PR | Title | Classification | Notes |
|----|-------|---------------|-------|
| #6103 | Initial support for analyzing and configuring integer dedicated columns | Docs present | `docs/sources/tempo/operations/tempo_cli.md` documents all new flags (`--num-int-attr`, `--int-percent-threshold`, `--include-well-known`, `int/` column prefix). `dedicated_columns.md` explains the 5% threshold heuristic. Despite unchecked documentation checkbox in PR, workspace docs are complete. |
| #5962 | Accept: application/llm | Docs update needed | `docs/sources/tempo/api_docs/_index.md` documents the LLM Accept header for trace lookup and tag value endpoints, but uses `application/vnd.grafana.llm` — the PR body specifies `application/llm`. Header name discrepancy requires engineering clarification. Additionally, `api_docs/mcp-server.md` does not mention that the MCP server automatically sets this header. |
| #5840 | build(deps): bump cloud.google.com/go/storage 1.56.0 → 1.57.1 | No docs required | Dependabot dependency bump with no user-facing behavior change. |

---

## Gap Summary (Prioritized for Phase 1.75 Work)

### 1. Docs update needed — Highest priority

**PR #5962 — Accept: application/llm**

Two issues require resolution before release notes can link to accurate documentation:

**Issue A: Header name discrepancy**
- The PR implements `Accept: application/llm`
- `docs/sources/tempo/api_docs/_index.md` documents `Accept: application/vnd.grafana.llm`
- These may be aliases or the header may have been renamed post-PR. **Engineering clarification required.**
- If the names differ, the API docs need to reflect the correct value used in production.
- Files to update: `docs/sources/tempo/api_docs/_index.md` (lines 219 and 623)

**Issue B: MCP server auto-sets header — undocumented**
- The PR notes: "MCP server automatically sets `application/llm` as the header."
- `docs/sources/tempo/api_docs/mcp-server.md` does not mention this behavior.
- Users configuring the MCP server should know the LLM-friendly format is used automatically.
- File to update: `docs/sources/tempo/api_docs/mcp-server.md`

**Suggested additions to `mcp-server.md`:**
Add a note that the MCP server automatically requests the LLM-optimized response format (`application/llm` or `application/vnd.grafana.llm` — confirm name) when calling trace and search endpoints, so users do not need to configure this manually.

---

### 2. Docs present — No action needed

**PR #6103 — Integer dedicated columns (analyse block)**

Documentation is complete across two files:
- `docs/sources/tempo/operations/tempo_cli.md` — all new CLI flags documented with defaults and examples
- `docs/sources/tempo/operations/dedicated_columns.md` — 5% prevalence threshold explained with user guidance

No additional documentation work needed. Link to these pages from release notes as appropriate.

---

### 3. No docs required — No action needed

**PR #5840 — Dependency bump**

Internal change. No release notes entry needed.

---

## Summary for Phase 1.75

| Priority | PR | Work required |
|----------|-----|---------------|
| P1 | #5962 | Engineering clarification on header name (`application/llm` vs `application/vnd.grafana.llm`); update `api_docs/_index.md` if needed; add MCP auto-header note to `mcp-server.md` |
| — | #6103 | No action — docs complete |
| — | #5840 | No action — no docs required |

**Blockers:** PR #5962 has an unresolved header naming discrepancy. Phase 1.75 writing cannot finalize the API docs section for this PR until engineering confirms the canonical Accept header value.
