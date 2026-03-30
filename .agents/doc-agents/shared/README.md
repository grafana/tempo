# Documentation Agent Shared Resources

This directory contains shared resources for documentation agents and writers working on Tempo documentation.

## Files Overview

### [`docs-context-guide.md`](docs-context-guide.md)
**Purpose**: Repo orientation for any Tempo documentation task

**When to use**:
- Before any doc task — writing, updating, or reviewing
- When you need to find where code maps to docs
- When verifying claims against the codebase
- When unsure about Tempo-specific conventions

**Key contents**:
- Architecture and component names
- Documentation layout and code-to-docs patterns
- Verification table (claim type → where to check)
- Tempo-specific conventions (naming, formatting)
- Gotchas that prevent common errors

### [`style-guide.md`](style-guide.md)
**Purpose**: Grafana documentation style guide and templates

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
- Multi-phase workflow (input gathering, documentation assessment, gap resolution, categorization, writing, validation, polish)
- Documentation assessment process for evaluating whether PRs need docs and whether those docs exist
- PR classification system (docs present, docs needed, docs update needed, no docs required)
- Document structure template with frontmatter and section layout
- Example prompts for common release notes tasks (initial draft, PR deep dive, documentation assessment, upgrade considerations)
- Tempo-specific style guidelines and conventions
- Iteration checklist for content completeness, documentation assessment, documentation coverage, and quality
- Feature-by-feature deep dive workflow for complex releases

### [`metrics-generator-knowledge.md`](metrics-generator-knowledge.md)
**Purpose**: Domain-specific knowledge about Tempo metrics-generator

**When to use**:
- Writing or updating metrics-generator documentation
- Understanding feature scope (span-metrics vs. service-graphs)
- Verifying configuration options
- Understanding user confusion points

**Key contents**:
- Feature scope (processor-specific vs. shared)
- Configuration structure examples
- Common user confusion points
- Version compatibility notes
- Key code file locations

## Usage Workflow

### For New Documentation

1. **Read** [`docs-context-guide.md`](docs-context-guide.md) for repo orientation
2. **Start with** [`style-guide.md`](style-guide.md) to understand formatting requirements
3. **Review** [`best-practices.md`](best-practices.md) for common patterns and pitfalls
4. **Reference** [`metrics-generator-knowledge.md`](metrics-generator-knowledge.md) if working on metrics-generator features
5. **Use** [`verification-checklist.md`](verification-checklist.md) before submitting

### For Release Notes

1. **Follow** [`release-notes-workflow.md`](release-notes-workflow.md) for the complete multi-phase process
2. **Use** [`docs-pr-check` skill](../../../.claude/skills/docs-pr-check/SKILL.md) for documentation assessment (Phase 1.5)
3. **Use** [`docs-pr-write` skill](../../../.claude/skills/docs-pr-write/SKILL.md) for documentation gap resolution (Phase 1.75)
4. **Reference** [`style-guide.md`](style-guide.md) for general style rules
5. **Use** [`verification-checklist.md`](verification-checklist.md) before submitting

### For Updating Existing Documentation

1. **Read** [`docs-context-guide.md`](docs-context-guide.md) for repo orientation
2. **Check** [`verification-checklist.md`](verification-checklist.md) to ensure all areas are covered
3. **Reference** [`best-practices.md`](best-practices.md) for common issues to avoid
4. **Verify** against [`metrics-generator-knowledge.md`](metrics-generator-knowledge.md) if updating metrics-generator docs
5. **Review** [`style-guide.md`](style-guide.md) for consistency

### For Addressing GitHub Issues

1. **Read** [`best-practices.md`](best-practices.md) section on "Addressing User Confusion"
2. **Check** [`metrics-generator-knowledge.md`](metrics-generator-knowledge.md) for known confusion points
3. **Follow** [`verification-checklist.md`](verification-checklist.md) to ensure fixes are complete
4. **Verify** against [`style-guide.md`](style-guide.md) for consistency

## Quick Reference

| Task | Primary Resource | Secondary Resource |
|------|-----------------|-------------------|
| Starting new docs | `docs-context-guide.md` | `style-guide.md` |
| Updating docs | `docs-context-guide.md` | `verification-checklist.md` |
| Writing release notes | `release-notes-workflow.md` | `style-guide.md` |
| PR docs assessment (triage) | `../../../.claude/skills/docs-pr-check/SKILL.md` | `release-notes-workflow.md` |
| PR docs writing (execution) | `../../../.claude/skills/docs-pr-write/SKILL.md` | `release-notes-workflow.md` |
| Metrics-generator work | `metrics-generator-knowledge.md` | `verification-checklist.md` |
| Fixing user issues | `best-practices.md` | `metrics-generator-knowledge.md` |
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

## Related Resources

- Main documentation: `docs/sources/tempo/`
- Configuration reference: `docs/sources/tempo/configuration/_index.md`
- Codebase: `modules/generator/processor/`
- CHANGELOG: `CHANGELOG.md`
- Skills workflow README: `.claude/skills/README.md`
- Skill: `.claude/skills/docs-workflow/SKILL.md`
- Skill: `.claude/skills/docs-pr-check/SKILL.md`
- Skill: `.claude/skills/docs-pr-write/SKILL.md`
- Skill: `.claude/skills/docs-review/SKILL.md`
