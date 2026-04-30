# Documentation context guide (foundation)

Read this before any documentation task — writing, updating, or reviewing — **after** you have loaded the repo's **local context** (see [`docs/project-context.md`](../../../docs/project-context.md)). This file describes **how** to orient in an unfamiliar docs repo; it does **not** list product-specific paths (those belong in local context).

## 1. Load local context first

Your repo (or team) should provide:

- `docs/project-context.md` — product identity, branches, doc root, code ↔ docs paths, frontmatter, validation paths
- Your project conventions file — `CLAUDE.md` (Claude Code), `.cursor/rules/` (Cursor), `.github/copilot-instructions.md` (VS Code / Copilot)

If none exist, ask the user for doc root, product name, and where architecture or "start here" docs live.

## 2. Orientation — where to look

- **Architecture and naming:** Read the product's architecture or overview page (path from **local context**). Use the same component names and terminology that page uses.
- **Documentation layout:** Map directories from local context (for example: reference under `configuration/`, tasks under `set-up/`, concepts under `introduction/`). Do not assume a fixed tree.
- **Code-to-docs patterns:** Local context should describe where config reference pages live, where API or query docs live, and how Helm or packaging docs are organized (if applicable).

## 3. How to verify what you write

| What you are verifying        | Where to check (from local context)      |
| ----------------------------- | ---------------------------------------- |
| Default values, flags       | Source code for the component (e.g. `config.go` or equivalent) |
| Option exists in docs       | Current doc tree on the target branch    |
| Version introduced          | Changelog or release notes source        |
| Syntax / examples           | Test fixtures or examples in code, if available |

Always validate claims against **code or generated reference**, not only PR descriptions.

## 4. Conventions

Only what is not already in `style-guide.md` (same directory):

- Match tone and depth of **sibling pages** in the same section (read 2–3 before writing).
- Use exact spelling for config keys and API fields as in source code.

### Required reading

- `style-guide.md` in this directory — always
- Subsystem-specific `AGENTS.md` or equivalent — when local context points you there

## 5. Common gotchas (generic)

- **Generated reference pages:** Some projects auto-generate API or manifest docs. Local context must say which files are generated and must not be hand-edited.
- **Legacy names in APIs:** Public parameters or paths may keep historical names while implementation changes. Prefer **code** as source of truth.
- **Defaults vs latest:** The newest format or version in code may not be the documented default. Confirm against the configuration or release docs your context identifies.
