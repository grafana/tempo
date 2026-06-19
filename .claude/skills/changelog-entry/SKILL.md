---
name: changelog-entry
description: Create or fix Tempo changelog entries with .chloggen. Use when a PR needs a changelog, CHANGELOG.md was edited directly, or the user asks to add/update/amend release notes.
allowed-tools: Bash Read Grep Edit
---

# Tempo Changelog Entry

Use this skill when adding, replacing, or fixing a Tempo changelog entry for a PR.

## Core Rule

Do not edit `CHANGELOG.md` directly for unreleased PR changes. Tempo manages `CHANGELOG.md` with `.chloggen`; every PR adds a small YAML file under `.chloggen/` instead.

Always read `.chloggen/README.md` before making changes if you have not already read it in the current session.

## Workflow

### 1. Check the worktree

Before editing, inspect the current state:

```bash
git status --short --branch
git diff --name-status
```

If there are unrelated user changes, leave them untouched. If they directly conflict with the changelog work, ask the user how to proceed.

### 2. Determine the entry details

Collect or infer:

- `change_type`: one of `breaking`, `change`, `feature`, `enhancement`, `bug_fix`, `security`
- `component`: must be listed in `.chloggen/config.yaml`
- `note`: short user-facing description without the component prefix
- `issues`: PR number(s), for example `[7288]`; leave as `[]` only when auto-fill at release is appropriate
- `user`: GitHub handle without `@`
- `subtext`: optional extra detail; leave blank when not needed

Use the change itself, PR metadata, or the old `CHANGELOG.md` line to infer these fields. Ask one short question only if the required fields cannot be inferred safely.

### 3. Generate or create the YAML file

Prefer the repo helper:

```bash
make chlog-new
```

If the current branch name is not suitable, or if on `main`, `master`, or a detached HEAD, pass an explicit filename:

```bash
make chlog-new FILENAME=<short-change-name>
```

Then fill the generated file. Minimal example:

```yaml
change_type: bug_fix
component: distributor
note: bound per-trace slice preallocation during rebatching.
issues: [7288]
subtext:
user: carles-grafana
```

Keep `note` concise. The rendered changelog prefixes the component, so write `note: bound ...`, not `note: distributor: bound ...`.

### 4. Replace direct CHANGELOG.md edits

If the branch edited `CHANGELOG.md` directly for an unreleased entry, remove that direct edit and keep the `.chloggen/*.yaml` entry instead.

Do not discard unrelated `CHANGELOG.md` changes unless they are clearly generated unreleased entries being replaced by `.chloggen` files. Inspect the diff first:

```bash
git diff -- CHANGELOG.md
```

For branches where `CHANGELOG.md` should match the base branch, restore only that file from the base after confirming the diff is only unreleased changelog content:

```bash
git restore --source origin/main -- CHANGELOG.md
```

### 5. Validate

Always validate the entry:

```bash
make chlog-validate
```

If useful, preview the rendered output:

```bash
make chlog-preview
```

`chlog-preview` can fail when `issues: []` and the PR number cannot be inferred yet. In that case, either set `issues` explicitly or report the limitation.

### 6. Stage and commit or amend when requested

Before committing or amending, inspect the staged and unstaged changes:

```bash
git status --short --branch
git diff --stat
git diff -- .chloggen CHANGELOG.md
git log --oneline -10
```

Stage only the intended files:

```bash
git add .chloggen/<entry>.yaml
git add CHANGELOG.md # only if removing a direct changelog edit
```

If the user asked to amend the PR commit:

```bash
git diff --cached --check
git commit --amend --no-edit
```

If the user asked for a new commit, use a concise message such as:

```bash
git commit -m "chore: add changelog entry"
```

Never push unless the user explicitly asks.

## Final Response

Report:

- The `.chloggen` file created or updated
- Whether `CHANGELOG.md` direct edits were removed
- Validation run and result
- Commit hash if a commit or amend was performed
- Any unrelated files left untouched
