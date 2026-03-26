# Doc agents

Use AI to help write, update, and review Tempo documentation. These agents and skills handle research, drafting, validation, and formatting. You stay in control — reviewing output, answering questions, and deciding when to submit.

## Getting started

Open this repo in your AI tool (Cursor, Claude Code, Copilot, etc.) and tell the agent what you need.

### I have PRs that need docs

Provide your PR numbers and let the agent check, write, and review:

```text
/docs-workflow

PRs: #6126, #5982, #6103
Branch: release-2.10
```

The agent runs three steps — triage, write, review — and stops between each one so you can review before it continues.

You can also run any step on its own:

| I want to...                     | Say this                                           |
|----------------------------------|----------------------------------------------------|
| Check if PRs need docs           | `/docs-pr-check` with a list of PR numbers         |
| Write docs for specific PRs      | `/docs-pr-write` with PR numbers and target branch |
| Review docs I already wrote      | `/docs-review` with the file paths to review       |

### I need to write docs from scratch

For new features or topics not tied to a specific PR:

```text
Run writer-agent.md using style-guide.md.

I need to document the new MCP server feature.
```

The agent walks you through five stages interactively: learn the feature, plan the structure, draft the content, review it, and prepare the PR. You can start at any stage.

### Quick reference

| Task | What to use |
|------|-------------|
| PRs shipped and need docs | `/docs-workflow` with PR numbers |
| Check a PR list for doc gaps | `/docs-pr-check` |
| Write docs for flagged PRs | `/docs-pr-write` |
| Review changed doc files | `/docs-review` |
| Write docs from scratch | `writer-agent.md` with `style-guide.md` |
| Create release notes | Follow `release-notes-workflow.md` |

## What you do (as the writer)

Regardless of which workflow you use, your responsibilities are:

- Answer questions from the agent
- Review drafts and outputs
- Approve or edit the content
- Decide when to commit and open a PR

The agent handles the rest: reading the repo context guide, looking up PRs, searching existing docs, checking the style guide, and validating claims against the codebase.

## Directory structure

```
.agents/
├── README.md                       ← you are here
└── doc-agents/
    ├── writers/
    │   └── writer-agent.md         # Full documentation workflow agent
    └── shared/
        ├── README.md               # Detailed guide for shared resources
        ├── docs-context-guide.md   # Repo orientation: code-to-docs mapping, conventions, gotchas
        ├── style-guide.md          # Grafana style rules and templates
        ├── best-practices.md       # Lessons learned and common pitfalls
        ├── verification-checklist.md  # Pre-submission quality checklist
        ├── release-notes-workflow.md  # Multi-phase release notes process
        └── metrics-generator-knowledge.md  # Domain knowledge for metrics-generator

.claude/skills/
├── README.md                       # Skills workflow overview
├── docs-workflow/SKILL.md          # End-to-end pipeline: check → write → review
├── docs-pr-check/SKILL.md          # Triage PR documentation status
├── docs-pr-write/SKILL.md          # Write/update docs for flagged PRs
└── docs-review/SKILL.md            # Review docs for style, accuracy, completeness
```

### Writer agent

[`doc-agents/writers/writer-agent.md`](doc-agents/writers/writer-agent.md) is the primary documentation workflow agent. It walks you through five stages: **Teacher → Information Architect → Author (new or update) → Reviewer → Committer**. You can run the full workflow or start at any stage.

### Shared resources

These files live in [`doc-agents/shared/`](doc-agents/shared/) and are used by agents, skills, and human writers. See the [shared README](doc-agents/shared/README.md) for detailed descriptions and usage workflows.

| File                                                                                 | Purpose                                                                                                      |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| [`docs-context-guide.md`](doc-agents/shared/docs-context-guide.md)                   | Repo orientation: code-to-docs mapping, key file paths, verification patterns, conventions, and gotchas      |
| [`style-guide.md`](doc-agents/shared/style-guide.md)                                 | Grafana documentation style rules, templates, and formatting requirements                                    |
| [`best-practices.md`](doc-agents/shared/best-practices.md)                           | Pre-writing checklist, common pitfalls, documentation patterns (for human writers)                           |
| [`verification-checklist.md`](doc-agents/shared/verification-checklist.md)           | Comprehensive pre-submission checklist for accuracy, consistency, and completeness                           |
| [`release-notes-workflow.md`](doc-agents/shared/release-notes-workflow.md)           | Multi-phase workflow for creating release notes, from CHANGELOG curation through final polish                |
| [`metrics-generator-knowledge.md`](doc-agents/shared/metrics-generator-knowledge.md) | Domain knowledge for metrics-generator: feature scope, config structure, common confusion points, v3 changes |

### Skills

Skills are invokable workflows that live in `.claude/skills/`. They perform specific tasks and can be used independently or as part of a larger workflow.

| Skill                                                                 | Invocation               | Purpose                                                                                                   |
| --------------------------------------------------------------------- | ------------------------ | --------------------------------------------------------------------------------------------------------- |
| [`docs-workflow`](../.claude/skills/docs-workflow/SKILL.md)           | `/docs-workflow`         | End-to-end pipeline: check PRs for doc gaps → write docs → review the result                              |
| [`docs-pr-check`](../.claude/skills/docs-pr-check/SKILL.md)           | `/docs-pr-check`         | Triage a list of PRs: classify each as docs present, docs needed, docs update needed, or no docs required |
| [`docs-pr-write`](../.claude/skills/docs-pr-write/SKILL.md)           | `/docs-pr-write`         | Write or update documentation for PRs flagged by `docs-pr-check`                                          |
| [`docs-review`](../.claude/skills/docs-review/SKILL.md)               | `/docs-review`           | Review documentation changes for style, accuracy, and completeness                                        |

The PR skills work as a three-step pipeline: check → write → review. Use `/docs-workflow` to run all three in sequence, or invoke each skill individually. Refer to the [skills README](../.claude/skills/README.md) for details on the handoff contract.

## Workflows

Choose the workflow that matches your task.

### General documentation (new or update) with writer-agent.md

Use the writer agent with shared resources for any documentation task that is not tied to a specific PR list or release.

1. Run the writer agent: _"Run `writer-agent.md` using `style-guide.md`."_
2. The agent walks you through each stage. Answer its questions, review its output, and decide when to advance.
3. Before submitting, review against [`verification-checklist.md`](doc-agents/shared/verification-checklist.md).

### PR-driven documentation

Use the PR workflow when you have a list of PRs that need documentation work (outside of a full release notes workflow).

**Recommended:** Run `/docs-workflow` with your PR list. It runs all three steps (check → write → review) in sequence and asks for your input between each step.

**Or run each step individually:**

1. Read [`docs-context-guide`](doc-agents/shared/docs-context-guide.md) for repo orientation.
2. Run `/docs-pr-check` with your PR list to classify documentation status.
3. Run `/docs-pr-write` on the PRs marked as needing docs.
4. Run `/docs-review` on the changed files for style, accuracy, and completeness.

### Release notes

Use the release notes workflow for creating per-version release notes. This is a multi-session process that combines shared resources and skills.

1. Follow [`release-notes-workflow.md`](doc-agents/shared/release-notes-workflow.md) — it covers the full process from CHANGELOG curation (Phase 0) through final polish (Phase 5).
2. At Phase 1.5, run `/docs-pr-check` to assess documentation status for each PR.
3. At Phase 1.75, run `/docs-pr-write` to fill documentation gaps.
4. Reference [`style-guide.md`](doc-agents/shared/style-guide.md) throughout for formatting and conventions.

### Metrics-generator documentation

When working on metrics-generator docs, load [`metrics-generator-knowledge.md`](doc-agents/shared/metrics-generator-knowledge.md) as additional context. It covers feature scope, configuration structure, common user confusion points, and v3 architectural changes.
