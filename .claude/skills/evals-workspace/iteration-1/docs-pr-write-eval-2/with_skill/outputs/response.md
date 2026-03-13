# docs-pr-write Evaluation: PR #6103

## Files Changed

1. `docs/sources/tempo/operations/dedicated_columns.md`
2. `docs/sources/tempo/configuration/_index.md`

## PR-to-Doc Mapping

| PR | What was documented | Where |
|----|---------------------|-------|
| #6103 — Initial support for analyzing and configuring integer dedicated columns | Expanded "Integer attribute selection" section with: storage cost rationale for the 5% threshold, CLI workflow using `--num-int-attr` and `--generate-cli-args`, annotated CLI output example, and a concrete `parquet_dedicated_columns` YAML example using `type: int` | `docs/sources/tempo/operations/dedicated_columns.md` |
| #6103 — configuration reference gap | Updated `parquet_dedicated_columns` type comment from `options: string` → `options: string, int` and scope comment from `options: resource, span` → `options: resource, span, event` in both the storage block section and the overrides section | `docs/sources/tempo/configuration/_index.md` |

## Detail of Changes

### `dedicated_columns.md`

Replaced the 4-line stub "Integer attribute selection" section with a full subsection (~45 lines) covering:

- **Why 5%**: Explained the storage cost model (4 bytes/value generic vs 0.35–1.15 bytes/value + 0.18 bytes/NULL in dedicated), so operators understand the heuristic rather than just applying a magic number.
- **CLI workflow**: Added a concrete `tempo-cli analyse block` example with `--num-int-attr` and `--generate-cli-args` flags.
- **CLI output example**: Included annotated sample output showing the integer attribute table with percentage-of-rows and recommendation markers (✅).
- **`--generate-cli-args` note**: Explained that this flag produces `int/`-prefixed arguments suitable for piping to `parquet convert-4to5`.
- **Configuration example**: Added a YAML block showing integer columns configured under `parquet_dedicated_columns` with `type: int` for span and resource scopes.
- **Closing guidance**: Reiterated that sparse attributes (< 5%) should stay in generic storage.

### `configuration/_index.md`

Two separate instances of `parquet_dedicated_columns` reference blocks corrected:

1. **Storage block section** (~line 1590): Updated `type` comment from `options: string` to `options: string, int`; updated `scope` comment from `options: resource, span` to `options: resource, span, event`; updated the count comment from "Up to 10 span attributes and 10 resource attributes" to "Up to 20 string attributes and 10 integer attributes per scope".
2. **Overrides section** (~line 2100): Same type, scope, and count updates applied.

## Open Items

- **Verify attribute limits against code**: The PR description does not explicitly state the per-scope integer column limit. The docs say 10 integer attributes per scope based on PR description context ("5 spare columns") — engineering confirmation recommended if this limit is formally specified in the codebase.
- **`--include-well-known` flag behavior**: The PR description says this flag is "default disabled for now" when migrating from vParquet4 to vParquet5. A note about when users should enable this flag in migration contexts could be added to the upgrade guide.
- **vParquet5 integer support**: The `dedicated_columns.md` configuration section at the top of the page mentions `vParquet5` support for array/blob but doesn't explicitly confirm integer column support in vParquet5 — this could be clarified.
