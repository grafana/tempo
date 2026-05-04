# Documentation agent shared resources

This directory contains **foundation** resources for documentation agents and writers. **Product-specific paths, repo layout, and naming** come from the repository's **local context** (`docs/project-context.md`).

## Files Overview

### [`docs-context-guide.md`](docs-context-guide.md)

**Purpose**: How to orient in a docs repo (after loading **local context**)

**When to use**:

- Before any doc task — writing, updating, or reviewing
- When you need to find where code maps to docs
- When verifying claims against the codebase
- When aligning with project-specific conventions (from local context)

**Key contents**:

- Load local context first; generic verification patterns
- Architecture and layout: **filled in by local context**, not duplicated here
- Gotchas pattern (generated pages, legacy names, defaults)

### [`style-guide.md`](style-guide.md)

**Purpose**: Documentation style guide and templates

**When to use**:

- Before writing any documentation
- When reviewing documentation for style compliance
- When unsure about formatting, phrasing, or structure

**Key contents**:

- Style rules (tense, voice, formatting)
- Template structures (concept, task, reference, scenario)
- Common style requirements

### [`best-practices.md`](best-practices.md)

**Purpose**: Best practices learned from documentation work

**When to use**:

- Starting a new documentation task
- Reviewing documentation before submission
- Learning from past documentation work
- Avoiding common pitfalls

**Key contents**:

- Pre-writing checklist
- Verification process
- Common pitfalls and solutions
- Documentation patterns (good vs. bad)
- Guidelines for addressing user confusion

### [`verification-checklist.md`](verification-checklist.md)

**Purpose**: Comprehensive checklist for verifying documentation accuracy

**When to use**:

- Before submitting documentation changes
- When updating existing documentation
- To ensure completeness and accuracy
- As a quality assurance step

**Key contents**:

- Codebase verification steps
- Configuration reference checks
- Version compatibility verification
- Style guide compliance
- Example quality checks

### [`release-notes-workflow.md`](release-notes-workflow.md)

**Purpose**: Workflow for creating Grafana Tempo release notes

**When to use**:

- Creating release notes for a new Tempo version
- Reviewing or updating existing release notes
- Understanding the multi-phase release notes process
- Looking up the document structure template or example prompts

**Key contents**:

- Multi-phase workflow (CHANGELOG curation, docs assessment, gap resolution, drafting, validation)
- PR classification (docs present, docs needed, docs update needed, no docs required)
- Templates and prompts
- Iteration checklist for quality

### [`personas.md`](personas.md)

**Purpose**: Persona and intent model for audience-fit checks

**When to use**:

- Matching content level to the target audience
- Running persona-check on documentation
- Deciding content type and depth

**Key contents**:

- Four personas (Learner, Practitioner, Expert, Operator)
- Use cases, entry states, interaction modes
- Content selection guidance

## Usage Workflow

### For New Documentation

1. **Read** [`docs-context-guide.md`](docs-context-guide.md) for repo orientation
2. **Start with** [`style-guide.md`](style-guide.md) to understand formatting requirements
3. **Review** [`best-practices.md`](best-practices.md) for common patterns and pitfalls
4. **Reference** `metrics-generator-knowledge.md` (in `.agents/doc-agents/shared/`) if working on metrics-generator features
5. **Use** [`verification-checklist.md`](verification-checklist.md) before submitting

### For Release Notes

1. **Follow** [`release-notes-workflow.md`](release-notes-workflow.md) for the complete multi-phase process
2. **Use** [`../docs-pr-check/SKILL.md`](../docs-pr-check/SKILL.md) for documentation assessment
3. **Use** [`../docs-pr-write/SKILL.md`](../docs-pr-write/SKILL.md) for documentation gap resolution
4. **Reference** [`style-guide.md`](style-guide.md) for general style rules
5. **Use** [`verification-checklist.md`](verification-checklist.md) before submitting

### For Updating Existing Documentation

1. **Read** [`docs-context-guide.md`](docs-context-guide.md) for repo orientation
2. **Check** [`verification-checklist.md`](verification-checklist.md) to ensure all areas are covered
3. **Reference** [`best-practices.md`](best-practices.md) for common issues to avoid
4. **Verify** against code-adjacent `AGENTS.md` or other subsystem docs when local context points you there
5. **Review** [`style-guide.md`](style-guide.md) for consistency

### For Addressing GitHub Issues

1. **Read** [`best-practices.md`](best-practices.md) section on "Addressing User Confusion"
2. **Follow** [`verification-checklist.md`](verification-checklist.md) to ensure fixes are complete
3. **Verify** against [`style-guide.md`](style-guide.md) for consistency

## Quick Reference

| Task | Primary Resource | Secondary Resource |
|------|-----------------|-------------------|
| Starting new docs | `docs-context-guide.md` | `style-guide.md` |
| Updating docs | `docs-context-guide.md` | `verification-checklist.md` |
| Writing release notes | `release-notes-workflow.md` | `style-guide.md` |
| PR docs assessment (triage) | `../docs-pr-check/SKILL.md` | `release-notes-workflow.md` |
| PR docs writing (execution) | `../docs-pr-write/SKILL.md` | `release-notes-workflow.md` |
| Metrics-generator work | `.agents/doc-agents/shared/metrics-generator-knowledge.md` | `verification-checklist.md` |
| Fixing user issues | `best-practices.md` | `verification-checklist.md` |
| Style questions | `style-guide.md` | `best-practices.md` |
| Pre-submission review | `verification-checklist.md` | `style-guide.md` |

## Maintenance

These files should be updated when:

- New patterns or best practices are discovered
- Common issues are identified and resolved
- Domain knowledge expands (new features, new confusion points)
- Style guide is updated
- Verification processes improve

## Contributing

When you discover new insights:

1. Document them in the appropriate file
2. Update the relevant sections
3. Add examples if helpful
4. Share with the team

## Related resources

- Doc tree, config reference, and codebase paths: `docs/project-context.md`
- CHANGELOG: `CHANGELOG.md` at repository root
- Skills README: [`../README.md`](../README.md)
- Skills: [`../docs-workflow/SKILL.md`](../docs-workflow/SKILL.md), [`../docs-pr-check/SKILL.md`](../docs-pr-check/SKILL.md), [`../docs-pr-write/SKILL.md`](../docs-pr-write/SKILL.md), [`../docs-review/SKILL.md`](../docs-review/SKILL.md)
