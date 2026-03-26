# Doc agents

We use two types of documentation agents: a generic AI agent, which serves as the default tool for product teams without a writer and handles the basic, standardized workflow; and the AI twin, a personalized agent that encodes your unique research, structure, writing, and review process. The generic agent fills the gap when no writer is available, while the AI twin amplifies your individual craft and raises the quality bar for the teams you support.

Both types of doc agent are designed to help you generate, update, and maintain documentation across any Grafana project. They guide you through the entire documentation workflow—from understanding the product to drafting, reviewing, and preparing PRs—or you can run them at any individual stage you choose. They build structure, create and update content, validate links, and surface issues, while you stay in control of what to approve, refine, or publish.

This README explains what each resource does, how to use them together, and what responsibilities remain with you as the writer.

## What you do (as the writer)

Regardless of which workflow you use, your responsibilities are:

- Choose which workflow and resources to use
- Answer questions from the agent
- Review drafts and outputs
- Approve or edit the content
- Decide when to commit and open a PR

The agents and skills handle research, drafting, structure, validation, and formatting automatically.

## Directory structure

```
.agents/
├── README.md                       ← you are here
└── doc-agents/
    ├── writers/
    │   └── writer-agent.md         # Full documentation workflow agent
    └── shared/
        ├── README.md               # Detailed guide for shared resources
        ├── style-guide.md          # Grafana style rules and templates
        ├── best-practices.md       # Lessons learned and common pitfalls
        ├── verification-checklist.md  # Pre-submission quality checklist
        ├── release-notes-workflow.md  # Multi-phase release notes process
        └── metrics-generator-knowledge.md  # Domain knowledge for metrics-generator

.claude/skills/
├── README.md                       # Skills workflow overview
├── docs-context-guide/SKILL.md     # Repo orientation for any doc task
├── docs-pr-check/SKILL.md          # Triage PR documentation status
└── docs-pr-write/SKILL.md          # Write/update docs for flagged PRs
```

### Writer agent

[`doc-agents/writers/writer-agent.md`](doc-agents/writers/writer-agent.md) is the primary documentation workflow agent. It walks you through five stages: **Teacher → Information Architect → Author (new or update) → Reviewer → Committer**. You can run the full workflow or start at any stage.

### Shared resources

These files live in [`doc-agents/shared/`](doc-agents/shared/) and are used by agents, skills, and human writers. See the [shared README](doc-agents/shared/README.md) for detailed descriptions and usage workflows.

| File                                                                                 | Purpose                                                                                                      |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| [`style-guide.md`](doc-agents/shared/style-guide.md)                                 | Grafana documentation style rules, templates, and formatting requirements                                    |
| [`best-practices.md`](doc-agents/shared/best-practices.md)                           | Pre-writing checklist, common pitfalls, documentation patterns (for human writers)                           |
| [`verification-checklist.md`](doc-agents/shared/verification-checklist.md)           | Comprehensive pre-submission checklist for accuracy, consistency, and completeness                           |
| [`release-notes-workflow.md`](doc-agents/shared/release-notes-workflow.md)           | Multi-phase workflow for creating release notes, from CHANGELOG curation through final polish                |
| [`metrics-generator-knowledge.md`](doc-agents/shared/metrics-generator-knowledge.md) | Domain knowledge for metrics-generator: feature scope, config structure, common confusion points, v3 changes |

### Skills

Skills are invokable workflows that live in `.claude/skills/`. They perform specific tasks and can be used independently or as part of a larger workflow.

| Skill                                                                 | Invocation               | Purpose                                                                                                   |
| --------------------------------------------------------------------- | ------------------------ | --------------------------------------------------------------------------------------------------------- |
| [`docs-context-guide`](../.claude/skills/docs-context-guide/SKILL.md) | Read before any doc task | Repo orientation: code-to-docs mapping, key file paths, verification patterns, and Tempo conventions      |
| [`docs-pr-check`](../.claude/skills/docs-pr-check/SKILL.md)           | `/docs-pr-check`         | Triage a list of PRs: classify each as docs present, docs needed, docs update needed, or no docs required |
| [`docs-pr-write`](../.claude/skills/docs-pr-write/SKILL.md)           | `/docs-pr-write`         | Write or update documentation for PRs flagged by `docs-pr-check`                                          |

The PR skills work as a two-stage pipeline: run `docs-pr-check` first to identify whether a PR needs doc (identify gaps), then run `docs-pr-write` on the flagged PRs to draft the documentation. Refer to the [skills README](../.claude/skills/README.md) for details on the handoff contract.

## Workflows

Choose the workflow that matches your task.

### General documentation (new or update) with writer-agent.md

Use the writer agent with shared resources for any documentation task that is not tied to a specific PR list or release.

1. Run the writer agent: _"Run `writer-agent.md` using `style-guide.md`."_
2. The agent walks you through each stage. Answer its questions, review its output, and decide when to advance.
3. Before submitting, review against [`verification-checklist.md`](doc-agents/shared/verification-checklist.md).

### PR-driven documentation

Use the PR skills when you have a list of PRs that need documentation work (outside of a full release notes workflow).

1. Read [`docs-context-guide`](../.claude/skills/docs-context-guide/SKILL.md) for repo orientation.
2. Run `/docs-pr-check` with your PR list to classify documentation status.
3. Run `/docs-pr-write` on the PRs marked as needing docs.
4. Review against [`verification-checklist.md`](doc-agents/shared/verification-checklist.md) before submitting.

### Release notes

Use the release notes workflow for creating per-version release notes. This is a multi-session process that combines shared resources and skills.

1. Follow [`release-notes-workflow.md`](doc-agents/shared/release-notes-workflow.md) — it covers the full process from CHANGELOG curation (Phase 0) through final polish (Phase 5).
2. At Phase 1.5, run `/docs-pr-check` to assess documentation status for each PR.
3. At Phase 1.75, run `/docs-pr-write` to fill documentation gaps.
4. Reference [`style-guide.md`](doc-agents/shared/style-guide.md) throughout for formatting and conventions.

### Metrics-generator documentation

When working on metrics-generator docs, load [`metrics-generator-knowledge.md`](doc-agents/shared/metrics-generator-knowledge.md) as additional context. It covers feature scope, configuration structure, common user confusion points, and v3 architectural changes.
