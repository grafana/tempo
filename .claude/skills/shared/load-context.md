# Load local context

Before running any skill, load the Tempo repository context:

1. Read `docs/project-context.md` from the workspace root. This file provides the product name, GitHub org/repo (`grafana/tempo`), docs root path (`docs/sources/tempo/`), branch naming conventions, code-to-docs mapping, validation paths, and known gotchas.
2. Substitute `grafana/tempo` and `docs/sources/tempo/` in commands and paths where required by the skill.

If `docs/project-context.md` is missing or unreadable, ask the user for the docs root path before proceeding.
