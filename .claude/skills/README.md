# Skills

AI skills for documentation and maintenance workflows in the Tempo repository. For technical writers, engineers, and community contributors who use Cursor, VS Code, or Claude Code.

## Available skills

| Skill | Command | What it does |
|-------|---------|--------------|
| PR triage | `/docs-pr-check` | Classify documentation status for each PR in a list |
| PR writing | `/docs-pr-write` | Write or update docs for PRs that need work |
| Doc review | `/docs-review` | Review docs for style, accuracy, and completeness |
| PR pipeline | `/docs-workflow` | Run check, write, and review end-to-end |
| Audience fit | `/persona-check` | Check whether content matches its intended audience |
| Vendor conflicts | `/fix-vendor-conflicts` | Resolve `vendor/` conflicts during a merge, rebase, or dependency upgrade |
| Go version update | `/update-go-version` | Update Go version across go.mod, Dockerfile, CI workflows, and tools |

## Set up

1. Install [Cursor](https://cursor.com/docs/skills), [VS Code with Copilot](https://code.visualstudio.com/docs/copilot/customization/agent-skills), or [Claude Code](https://code.claude.com/docs/en/skills)
2. For docs skills: Install [GitHub CLI](https://cli.github.com/) and authenticate (`gh auth login`)
3. Clone the repository and open it in your IDE
4. Skills are discovered automatically from `.claude/skills/`

### Project context

The doc skills use [`docs/project-context.md`](../../docs/project-context.md) to understand this repository. That file provides the product name, GitHub org/repo, doc root path, code-to-docs mapping, frontmatter conventions, validation paths, and known gotchas. Skills load it automatically through `shared/load-context.md` — you don't need to reference it in prompts.

If you're working on docs manually (not through a skill), read [`docs/project-context.md`](../../docs/project-context.md) for Tempo conventions and `shared/docs-context-guide.md` for general orientation.

If you're adapting these skills for another repository, copy `docs/project-context.md` and update the values for your product. The skills read from that file at runtime, so everything product-specific stays in one place.

## Docs skills

Use the docs skills to create or update documentation, review it for quality, or check whether it fits the right audience. Each skill works on its own, or you can run the PR pipeline to do all three in sequence.

- `/docs-pr-check`, `/docs-pr-write`, and `/docs-review` form a pipeline for PR-driven docs work. `/docs-workflow` runs all three steps end-to-end.
- `/persona-check` assesses whether a doc page fits its intended audience — not a style check (use `/docs-review` for that).

Skills draft and review content but never commit, push, or publish on their own. You decide when to advance each step, what to keep, and when to submit.

### PR workflow

Use `/docs-workflow` to run the full pipeline, or invoke each step individually. Provide a list of PR numbers and a target branch. The workflow runs check → write → review and asks for your input between each step.

1. **Triage** — `/docs-pr-check`. Provide a curated PR list. Returns a classification table (`Docs present`, `Docs needed`, `Docs update needed`, `No docs required`) and a prioritized gap summary.

2. **Write** — `/docs-pr-write`. Takes PRs marked `Docs needed` or `Docs update needed` from step 1, plus branch/version context. Produces updated docs files, a PR-to-doc mapping, and flags blockers needing engineering input.

3. **Review** — `/docs-review`. Takes the file paths changed in step 2 and any open items. Returns issues grouped by file and a summary.

### Automations

To run this pipeline unattended (scheduled or PR-triggered) as Cursor Automations against `grafana/tempo`, refer to [`../../.agents/doc-agents/cursor-automations-docs-workflow.md`](../../.agents/doc-agents/cursor-automations-docs-workflow.md). This is maintainer tooling for setting up cloud automations; it is not required for running the skills locally.

## Other skills

- *Vendor conflicts* — `/fix-vendor-conflicts`. Resolve `vendor/` directory conflicts during a merge, rebase, or dependency upgrade on main or release branches.

- *Go version update* — `/update-go-version`. Update the Go version across all relevant files: `go.mod`, `tools/go.mod`, Dockerfile, CI workflows, and the tools image tag.

## Shared resources

These files are loaded by skills automatically. This section is for maintainers editing or extending skills.

| File | Purpose |
|------|---------|
| `.claude/skills/shared/style-guide.md` | Documentation style rules, templates, and formatting |
| `.claude/skills/shared/verification-checklist.md` | Pre-submission checklist for accuracy and completeness |
| `.claude/skills/shared/best-practices.md` | Pre-writing checklist and common pitfalls |
| `.claude/skills/shared/release-notes-workflow.md` | Multi-phase workflow for release notes |
| `.claude/skills/shared/docs-context-guide.md` | General repo orientation for doc tasks |
| `.claude/skills/shared/load-context.md` | Instructions for loading local project context |
| `.claude/skills/shared/personas.md` | Persona and intent model for audience-fit checks |
| `.claude/skills/shared/handling-pr-content.md` | Rules for treating PR content as untrusted input |
