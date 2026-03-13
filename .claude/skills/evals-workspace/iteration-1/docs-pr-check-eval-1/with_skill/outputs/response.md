# docs-pr-check: PR #4905

## Classification Table

| PR | Title | Classification | Notes |
|----|-------|---------------|-------|
| #4905 | `{.a = nil }` | Docs present | `docs/sources/tempo/traceql/construct-traceql-queries.md` updated in the PR with 6 additions covering nil syntax; existing docs include a "Nil" subsection with examples and comparison operator examples using `nil` |

## Assessment Details

**PR Summary**: PR #4905 adds support for querying `nil` values in TraceQL. It introduces a new iterator and predicate for handling "null" values so users can check whether span attributes are missing or null using syntax like `{ span.attribute = nil }` or `{ span.attribute != nil }`.

**Documentation checklist**: ✅ "Documentation added" is checked in the PR body.

**Files changed under `docs/`**: ✅ Yes — `docs/sources/tempo/traceql/construct-traceql-queries.md` (6 additions, 0 deletions).

**Existing docs coverage**: The current workspace version of `construct-traceql-queries.md` contains:
- A dedicated **"Nil"** subsection (under "Value types and literals") explaining `nil` usage with two examples:
  ```
  { span.optional_field = nil }
  { span.required_field != nil }
  ```
- Multiple comparison operator examples using `nil`:
  - `{ span.any_attribute != nil }` — find spans where attribute exists
  - `{ span.any_attribute = nil }` — find spans where attribute is missing
  - `{ resource.service.version = nil }` — resource-level nil check
  - `{ event.exception.message = nil }` — event-scope nil check

**Completeness assessment**: Documentation is thorough. It covers:
- What `nil` means (missing or null attribute)
- Both `= nil` and `!= nil` operator forms
- Examples across span, resource, and event scopes

## Gap Summary

No documentation gaps identified. PR #4905 is fully documented.

1. ~~PRs where docs are entirely missing~~ — N/A
2. ~~Existing pages that need updates~~ — N/A
3. ~~PRs where doc completeness is uncertain~~ — N/A

**Action**: Link to `docs/sources/tempo/traceql/construct-traceql-queries.md#nil` in release notes.
