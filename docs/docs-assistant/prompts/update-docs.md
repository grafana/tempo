<!-- Agent prompt for the docs-update workflow. Works unmodified for most repos. -->

All context is in ONE file: `/tmp/claude-context.txt`
Read it now with the Read tool. It contains:

- The PR diff
- Docs writing guide (`docs/docs-assistant/prompts/docs-assistant.md`)
- Coverage plan (`docs/docs-assistant/context.yaml`)
- Docs file listing

Do NOT use Bash to read files or list directories. Use Read, Glob, or LS tools only.
Do NOT use Bash to write files. Use the Write tool only.
Do NOT explore the repository beyond `docs/`.
Do NOT run `gh pr diff` or read source files not in the diff.

## Step 1 — Decide

Based on the diff in `/tmp/claude-context.txt`, decide whether this PR introduces or changes user-facing behaviour covered by `docs/docs-assistant/context.yaml`.

If no docs changes are needed, output: **No docs changes required.** Then stop.

## Step 2 — Update docs

- Limit edits to at most 4 files (including `docs/docs-assistant/context.yaml`)
- Update `docs/docs-assistant/context.yaml` when the PR adds or materially changes a user-facing capability
- Update or create files in `docs/sources/` for user-facing behaviour only
- Create a new article only when the PR adds a feature not covered by any existing article
- Follow the docs writing guide
- Do not edit `docs/agent.md`
- Do not create commits or GitHub comments

## Step 3 — Finish

Write a 2–3 sentence summary to `/tmp/docs-agent-summary.txt` describing what changed and why.
Output: **Docs update complete.**

## Rules

- Only edit `docs/docs-assistant/context.yaml` and files under `docs/sources/`
- Maximum 4 files edited per run
- Keep changes tightly scoped to the PR's user-facing impact
