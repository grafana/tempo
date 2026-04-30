# Validate documentation claims against code

Supporting material for **Step 5** of `docs-pr-write`. Load this when verifying technical claims (skip for purely editorial changes with no behavioral or API assertions).

## 1. Scope: PR-first, never whole-repo

- Use the PR's **changed file list** as the primary map of where to look. Search only in the **target product repo** unless local context says otherwise ([`../../shared/load-context.md`](../../shared/load-context.md)).
- **Do not** scan or read the entire codebase. Verification is **PR-scoped**: start with files changed in the PR, then follow only **imports, re-exports, and shared constants/types** those files reference. Add schema/proto/Helm paths when local context maps them to this change.
- Use **targeted** search (for example `rg` limited to directories from the PR file list), not open-ended exploration. If a claim cannot be confirmed without spelunking unrelated packages, add an **open item** instead of widening the search.

## 2. Anchor with local context (large monorepos)

If `docs/project-context.md` (or equivalent) defines **Code validation paths** or **Code ↔ documentation mapping**, use those tables first when they overlap the PR's changes. Prefer directories and files listed there over inventing search roots.

## 3. What to verify where

| Kind of claim | Default source |
| -------------- | -------------- |
| APIs, config, server behavior | Source for field names, defaults, enums; schema/proto for API paths; Helm/YAML for config keys |
| UI and workflows (routes, labels, menus, dialogs, toggles) | **Frontend source** from the PR's changed files first, then the narrow dependency chain in §1. Do not infer visible UI copy from backend-only code. |

If PR description and code conflict, **code wins**; record the mismatch as an open item.

## 4. Live UI and visuals

If a claim is only fully verifiable in a running app (layout, visuals, interaction order), treat it as an **open item** unless the user provides an instance URL and you can verify with Browser DevTools MCP. Otherwise defer to screenshot-check or QA.
