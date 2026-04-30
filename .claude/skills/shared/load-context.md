# Load local context

Before running any skill, load your repository's local context:

1. Look for a local context file in the workspace. Check these paths in order and use the first match found:
   - `docs/project-context.md` or `project-context.md` (project root)
   - Tool-specific fallbacks: `.cursor/project-context.md`, `.github/project-context.md`
   - Legacy names: `docs/docs-mapping.md`, `docs/repo-context.md`, `docs-mapping.md`, `repo-context.md`

   This file provides the GitHub org/repo, docs root path, branch names, code-to-docs mapping, and validation paths for this repository.
2. Read `../shared/docs-context-guide.md` for generic repo orientation.
3. Substitute `YOUR_ORG/YOUR_REPO` and `<your-docs-root>` in commands and paths with the values from local context. If no local context file exists, ask the user for the GitHub org/repo and documentation root path before proceeding.
