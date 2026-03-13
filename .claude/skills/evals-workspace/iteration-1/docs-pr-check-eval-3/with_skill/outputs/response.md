# docs-pr-check: PRs #6103, #5962, #5840

## Classification Table

| PR | Title | Classification | Notes |
|----|-------|----------------|-------|
| #6103 | Initial support for analyzing and configuring integer dedicated columns | Docs update needed | Flags (`--num-int-attr`, `--int-percent-threshold`, `--generate-cli-args`) are listed in `operations/tempo_cli.md`, but the new integer heuristic logic (5% row-prevalence threshold, output format showing count/percentage/recommendations) is not explained. Documentation checkbox unchecked in PR. |
| #5962 | Accept: application/llm | Docs present | `api_docs/_index.md` documents `Accept: application/vnd.grafana.llm` for Query V2 and tag values endpoints. `api_docs/mcp-server.md` covers the MCP server context. Note: docs use header name `application/vnd.grafana.llm`; PR title says `application/llm` — verify header name accuracy. Documentation checkbox unchecked in PR. |
| #5840 | build(deps): bump cloud.google.com/go/storage from 1.56.0 to 1.57.1 | No docs required | Dependabot dependency bump. Pure internal change with no user-facing impact. |

---

## Gap Summary

### Priority 1 — Docs update needed (user-facing behavior not documented)

**PR #6103 — `tempo-cli analyse block`: integer dedicated column heuristics**

- **File to update:** `docs/sources/tempo/operations/tempo_cli.md`
- **What's present:** Flags `--num-int-attr`, `--int-percent-threshold`, `--generate-cli-args`, `--include-well-known` are listed for both `analyse block` and `analyse blocks` commands.
- **What's missing:**
  - No explanation of the 5% row-prevalence heuristic for integer columns (why 5%, what it means)
  - No description of the new output format: "Total integer attribute values", "Top N integer attributes by count", percentage of values vs. percentage of rows, "✅ Recommended dedicated column" indicator
  - No example showing integer analysis output
  - No explanation of how `--generate-cli-args` output integrates with the `parquet convert-4-to-5` command for integer columns
  - The intro text ("Analyses a block and outputs a summary of the block's generic attributes") doesn't reflect that the command now analyzes integer attribute columns in a distinct way from string attributes

### Priority 2 — Verification needed (possible header name discrepancy)

**PR #5962 — `Accept: application/llm` / `application/vnd.grafana.llm`**

- **Files that cover this:** `docs/sources/tempo/api_docs/_index.md` (Query V2 section and tag values section), `docs/sources/tempo/api_docs/mcp-server.md`
- **Documentation appears present**, but the PR title references `application/llm` while the docs document `application/vnd.grafana.llm`.
- **Action needed:** Confirm with the PR author which header name is canonical. If the final implementation uses `application/vnd.grafana.llm`, docs are accurate and no further action is needed. If the implementation uses `application/llm`, the docs need to be corrected.
- The MCP server page does not explicitly state that the MCP server automatically sets the `Accept: application/vnd.grafana.llm` header — this behavioral note from the PR could be added.

### Priority 3 — No action needed

**PR #5840 — Dependency bump**

No documentation work required.
