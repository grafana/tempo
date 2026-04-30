# Release notes workflow

Instructions for creating release notes for Grafana Tempo releases. This workflow is tool-agnostic and works with any AI coding assistant that can access the GitHub API.

## Role

Act as an experienced technical writer and distributed tracing expert for Grafana Labs.

Write release notes for software developers, SREs, and platform engineers who use Tempo for distributed tracing and observability.

Focus on user impact, practical examples, and clear upgrade guidance.

## Before you begin

Ensure you have the following:

- Access to the GitHub API to look up PR details (for example, GitHub MCP server, `gh` CLI, or direct API access)
- Access to the Tempo repository (`grafana/tempo`) and its PRs
- Previous release notes for format reference (for example, `v2-10.md`)
- Source material: curated PR list from the engineering team OR CHANGELOG unreleased section

## Workflow phases

### Phase 0: Source curation (human-driven, AI-assisted)

The CHANGELOG must be sorted and grouped before drafting begins. An AI assistant can do the heavy lifting -- reading each PR, understanding context, and proposing groupings -- but a human must review, adjust, and approve the final result. Do not proceed to Phase 1 until this is complete.

The goal of this phase is to understand the shape of the release: what the major feature areas are, which CHANGELOG entries belong to each area, and what to include or exclude from the release notes.

#### Step 1: Obtain the raw CHANGELOG

Get the raw CHANGELOG or PR list for the upcoming release.

#### Step 2: Identify known headline features

Before sorting the CHANGELOG, ask: **are the main features of this release already known?**

The engineering team, product manager, or writer may already know which features define this release. If so, these known headline features and their associated PRs become the primary emphasis of the release notes. Everything else in the CHANGELOG gets organized around them.

If the main features are known:
- List them along with the key PRs associated with each feature
- These features become the top-level feature areas for sorting
- The AI-assisted sorting in Step 3 should group remaining CHANGELOG entries around these known areas and identify any additional areas the team may not have mentioned

If the main features aren't known yet:
- Proceed to Step 3 and let the AI propose the major feature areas by reading through the CHANGELOG entries
- Review the AI's proposed areas with the engineering team in Step 4 to confirm

#### Step 3: AI-assisted sorting

Provide the raw CHANGELOG to the AI assistant, along with the known headline features if available. Ask it to:

1. **Look up each PR** to read the description, understand the change, and assess user impact.

2. **Identify or confirm the major feature areas** for this release:
   - If headline features were provided in Step 2, use those as the primary feature areas and group associated PRs under them. Then identify any additional areas from the remaining entries.
   - If no headline features were provided, identify the major feature areas from the CHANGELOG entries themselves. These aren't fixed categories -- they emerge from the entries. The AI should read through all entries and propose the themes that define the release.

   For example, the Tempo 2.10 release had these major feature areas:
   - TraceQL (new query capabilities like `= nil`, `span:childCount`, `minInt`/`maxInt`)
   - Metrics-generator (entity-based limiting, overflow series, cardinality estimation)
   - vParquet5 (production readiness, dedicated columns, blob detection)
   - LLM-optimized API responses
   - User-configurable overrides
   - Project Rhythm (new architecture, experimental)

   A different release might have entirely different major areas, for example, query performance, multi-tenancy, or new storage backends. Let the CHANGELOG entries tell you what the release is about.

3. **Group every CHANGELOG entry** into the feature areas:
   - Assign each entry to the feature area where it has the most user impact. PRs associated with known headline features should be grouped under those areas first.
   - Within each group, weight the entries: identify which are headline items versus supporting changes, minor improvements, or bug fixes.
   - Flag entries that should be excluded from release notes (internal refactors, test-only changes, dependency bumps, CI/CD changes).
   - Flag entries that are uncertain -- these need human judgment.

4. **Return a structured, grouped list** organized by feature area, with entries ordered by importance within each group and exclusions listed separately. Known headline features should appear first.

#### Step 4: Human review

The AI's proposed grouping is a starting point, not the final answer. Review the grouped list and:

- Adjust feature area names and boundaries (the AI may split or merge areas incorrectly)
- Move entries between groups if the AI misjudged the primary feature area
- Override include/exclude decisions where the AI lacked context
- Resolve uncertain entries
- Confirm which feature areas deserve featured sections versus brief mentions

#### Step 5: Coordinate with the engineering team

Review the grouped list with the engineering team to:
- Confirm the major feature areas are correct and complete
- Identify the top 3-5 features for the introduction highlights
- Identify breaking changes requiring upgrade documentation
- Flag features that need documentation updates
- Resolve any remaining entries you're unsure about

#### Step 6: Produce the curated input

The final output of Phase 0 is a curated, grouped PR list organized by major feature area, with entries weighted by importance within each group. This list is the input for all subsequent phases.

Do not begin Phase 1 until this curated list has been reviewed and approved by the writer or team lead.

### Phase 1: Input gathering

Using the curated PR list from Phase 0:

1. **Look up each PR** using the GitHub API:
   - Read PR descriptions for context and user impact
   - Check linked issues for additional context
   - Identify configuration changes, new flags, or migration steps
   - **Check the PR checklist**: Note if "Documentation added" is checked
   - Look for documentation files changed in the PR

### Phase 1.5: Documentation assessment

For each PR, evaluate whether it needs documentation and whether that documentation exists. This is a critical review step that prevents features from shipping without adequate docs.

Run `/docs-pr-check`. See [`.claude/skills/docs-pr-check/SKILL.md`](../../../.claude/skills/docs-pr-check/SKILL.md) for the full classification process, criteria, and return format.

### Phase 1.75: Documentation gap resolution

For each PR classified as "docs needed" or "docs update needed", create or update the required documentation pages.

Run `/docs-pr-write`. See [`.claude/skills/docs-pr-write/SKILL.md`](../../../.claude/skills/docs-pr-write/SKILL.md) for the full execution steps, validation process, and return format.

### Phase 2: Categorization

Map the grouped feature areas from Phase 0 into the release notes document structure. The major feature areas identified during curation become the featured sections and topic groupings in the release notes.

Group entries into these document sections (in order):

1. **Introduction highlights** - 3-5 bullet points drawn from the headline items across the major feature areas
2. **Featured sections** - Each major feature area with enough substance gets its own h2 section with deep dives and examples
3. **Features and enhancements** - Remaining entries grouped by their feature area under a shared h2
4. **Upgrade considerations** - Breaking changes, deprecations, migration steps (may span multiple feature areas)
5. **Bug fixes** - Brief list with PR links
6. **Security fixes** - If applicable

The topic groupings within "Features and enhancements" should use the major feature areas from Phase 0, not a fixed list. Every release has different feature areas. For example, Tempo 2.10 used TraceQL, metrics-generator, vParquet5, user-configurable overrides, and others. A future release might use entirely different groupings.

### Phase 3: Writing each entry

For each entry:

1. **Summarize in 2-3 sentences** focusing on user impact, not implementation details
2. **Include examples** where they clarify usage:
   - TraceQL queries for query features
   - Configuration snippets for new options
   - Before/after for migrations
3. **Link to the PR** at the end: `[[PR XXXX](https://github.com/grafana/tempo/pull/XXXX)]`
4. **Link to documentation** if it exists: `[documentation](/docs/tempo/<TEMPO_VERSION>/path/to/doc/)`

For upgrade considerations:

1. Explain what changed and why it matters
2. Describe who is affected
3. Provide migration steps or configuration changes
4. Include code examples for configuration updates

### Example prioritization

When multiple examples are possible, prioritize based on:

1. **Practical debugging value**: Does this help users solve real problems?
2. **Common use cases**: Will most users need this?
3. **Unique capability**: Does this show something only possible with this feature?

### Updating existing documentation

Major features often require updates beyond release notes:

1. **Query documentation** (`traceql/construct-traceql-queries.md`): Add examples for new intrinsics or functions and update intrinsic tables with new entries.

2. **Configuration reference** (`configuration/_index.md`): Document new configuration options and add examples showing how to enable features.

3. **Operations documentation**: Document new metrics and upgrade paths for breaking changes.

For each major PR, ask: what existing documentation pages need to be updated beyond the release notes?

### Phase 4: Code validation

Before finalizing examples, validate them against the codebase:

1. **TraceQL validation**:
   - Search `pkg/traceql/test_examples.yaml` for similar valid patterns
   - Check `pkg/traceql/ast.go` to verify intrinsic types (TypeInt, TypeString, etc.)
   - Confirm operators are valid for the data type

2. **Configuration validation**:
   - Search for the config option in `modules/` or check `configuration/_index.md`
   - Verify YAML structure matches actual implementation

### Phase 5: Final polish

1. Verify all PR links are correct and accessible
2. Check that configuration examples match the actual code
3. Ensure documentation links use `<TEMPO_VERSION>` placeholder
4. Run linter to fix style issues
5. Apply sentence case to all headings
6. Remove CHANGELOG artifacts (usernames, brackets around categories)

## Patch releases (X.Y.Z)

Patch releases (for example, 2.10.1) are maintenance releases. **Update the existing** X.Y release notes file (`v2-10.md`); do not create a new file. Patch releases may include bug fixes, Go version changes, CVE or security patches, and other changes. Use the GitHub release page as the source of truth (for example, `https://github.com/grafana/tempo/releases/tag/v2.10.1`).

### Sections to update

#### Security fixes

If the patch addresses CVEs or security vulnerabilities:

- Add or update the **## Security fixes** section (place it before Bug fixes).
- Add a `### X.Y.Z` subsection.
- For each fix: describe what was updated (Go version, dependency, etc.), link to CVE advisories, and include PR links.

**Example from v2.9.1:**

```markdown
## Security fixes

The following updates were made to address security issues.

### 2.9.1

- Updated Go to version 1.25.5 to address [CVE-2025-61729](https://github.com/advisories/GHSA-7c64-f9jr-v9h2), [CVE-2025-47907](...), ... [[PR 6089](...), [PR 6096](...)]
- Updated `golang.org/x/crypto` to address [CVE-2025-47914](...), [CVE-2025-58181](...). [[PR 6235](...)]
- Updated `github.com/expr-lang/expr` to v1.17.7 to address [CVE-2025-68156](...). [[PR 6234](...)]
```

#### Upgrade considerations

For **Changes** that affect users but are not security-related (for example, a Go version bump without CVEs):

- Update the relevant subsection under **Upgrade considerations**.
- For Go upgrades: update the existing "Go version upgrade" bullet with the new version and PR links.

#### Bug fixes

- Add a `### Version X.Y.Z` subsection under **## Bug fixes**, placed above the existing entries.
- List each bugfix as a bullet with a brief user-focused description and PR link: `- [Description]. [[PR XXXX](URL)]`

### Workflow

1. Open the GitHub release page for the patch.
2. **Security fixes**: If CVEs or security fixes are present, add or update **## Security fixes** with a version subsection. Include CVE links (GitHub advisories) and PR links.
3. **Upgrade considerations**: For non-security changes (for example, Go version), update the relevant content under Upgrade considerations.
4. **Bug fixes**: Add a `### Version X.Y.Z` subsection and list each bugfix with PR links.
5. **Quality checks**: Verify all security fixes, changes, and bugfixes are documented with correct links.

### Content routing

| Content type | Section | Example |
|--------------|---------|---------|
| CVE fixes, security patches | Security fixes | v2.9.1: Go 1.25.5, golang.org/x/crypto, expr |
| Go or dependency upgrade for security | Security fixes | Include CVE advisory links and PR links |
| Go or dependency upgrade (non-security) | Upgrade considerations | Update "Go version upgrade" bullet |
| Bug fixes | Bug fixes | Version subsection with PR links |
| Other notable changes | Upgrade considerations or brief note | Case by case |

### Differences from full releases

| Full release (X.Y) | Patch release (X.Y.Z) |
|--------------------|------------------------|
| New release notes file | Update existing file |
| Phase 0–5 curation and drafting | Directly add content from GitHub release |
| New featured sections, examples | Add Security fixes, Bug fixes, and changes only |
| Multi-session workflow | Single, additive update |

## Document structure

Use this template for new release notes:

```markdown
---
title: Version X.Y release notes
menuTitle: VX.Y
description: Release notes for Grafana Tempo X.Y
weight: 10
---

# Version X.Y release notes

<!-- vale Grafana.We = NO -->
<!-- vale Grafana.GoogleWill = NO -->
<!-- vale Grafana.Timeless = NO -->
<!-- vale Grafana.Parentheses = NO -->

The Tempo team is pleased to announce the release of Tempo X.Y.

This release gives you:

- [Highlight 1 - brief description of major feature]
- [Highlight 2 - brief description of major feature]
- [Highlight 3 - brief description of major feature]
- [Highlight 4 - optional]

These release notes highlight the most important features and bug fixes. For a complete list, refer to the [Tempo changelog](https://github.com/grafana/tempo/releases).

## [Featured section title]

[2-3 paragraphs explaining the feature, its benefits, and use cases]

[TraceQL or configuration example if applicable]

### [Subsection if needed]

[Additional details]

## Features and enhancements

The most important features and enhancements in Tempo X.Y are highlighted below.

### [Topic area]

[Grouped entries with descriptions and PR links]

## Upgrade considerations

When [upgrading](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) to Tempo X.Y, be aware of these considerations and breaking changes.

### [Breaking change title]

[Description of what changed, who is affected, and migration steps]

## Bug fixes

For a complete list, refer to the [Tempo CHANGELOG](https://github.com/grafana/tempo/releases).

- [Brief description of fix]. [[PR XXXX](https://github.com/grafana/tempo/pull/XXXX)]
```

## Example prompts

These prompts work with any AI coding assistant. Replace `[look up the PR]` with whatever method your tool supports (GitHub MCP, `gh` CLI, API calls, etc.).

### CHANGELOG curation assist (Phase 0)

Use these prompts to help with the sorting process, but **a human must review and approve the final grouped list** before proceeding.

#### Sort CHANGELOG with known headline features

Use this prompt when the main features of the release are already known. The known features become the primary emphasis, and the remaining entries are organized around them.

> Here is the unreleased section of the CHANGELOG for Tempo vX.Y. The main features of this release are already known:
>
> - [Feature 1]: [brief description, key PRs if known]
> - [Feature 2]: [brief description, key PRs if known]
> - [Feature 3]: [brief description, key PRs if known]
>
> Sort the CHANGELOG entries using these headline features as the primary feature areas. For each CHANGELOG entry:
>
> 1. Look up the PR to read the description and understand the change
> 2. Assess the user impact
>
> Then:
>
> 1. **Group entries under the known headline features first.** PRs associated with each headline feature should be grouped together, with the key PRs as the headline items.
> 2. **Identify any additional feature areas** from the remaining entries that don't fit under the known features.
> 3. **Order entries within each group** by importance: headline items first, then supporting changes, then minor fixes.
> 4. **Flag entries to exclude** from release notes (internal refactors, test-only changes, dependency bumps, CI/CD changes). List these separately with a brief reason.
> 5. **Flag uncertain entries** that need human judgment. Explain why you're unsure.
>
> Return the result as a structured, grouped list with the known headline features listed first. I'll review and adjust your groupings before we proceed.
>
> [Paste raw CHANGELOG section here]

#### Sort CHANGELOG without known headline features

Use this prompt when the main features aren't known yet. The AI proposes the major feature areas by reading through the CHANGELOG.

> Here is the unreleased section of the CHANGELOG for Tempo vX.Y. I need you to sort these entries into major feature areas for the release notes.
>
> For each CHANGELOG entry:
>
> 1. Look up the PR to read the description and understand the change
> 2. Assess the user impact (is this a headline feature, a supporting enhancement, a minor fix, or an internal-only change?)
>
> Then:
>
> 1. **Identify the major feature areas** for this release. These should be the themes that emerge from the entries themselves (for example, "TraceQL enhancements", "metrics-generator cardinality management", "vParquet5 production readiness"). Don't use fixed categories; let the entries tell you what this release is about.
> 2. **Group every entry** into the proposed feature areas. Assign each entry to the area where it has the most user impact.
> 3. **Order entries within each group** by importance: headline items first, then supporting changes, then minor fixes.
> 4. **Flag entries to exclude** from release notes (internal refactors, test-only changes, dependency bumps, CI/CD changes). List these separately with a brief reason.
> 5. **Flag uncertain entries** that need human judgment. Explain why you're unsure.
>
> Return the result as a structured, grouped list organized by feature area. I'll review and adjust your groupings before we proceed.
>
> [Paste raw CHANGELOG section here]

#### Adjust groupings after human review

Use this prompt after reviewing the AI's initial sort, if you need to refine the groupings.

> I've reviewed your proposed feature area groupings. Here are my adjustments:
>
> [Describe adjustments: moved entries, renamed areas, changed include/exclude decisions, etc.]
>
> Please update the grouped list with these changes and return the revised version.

### Initial draft from curated PR list

This prompt requires a curated PR list from Phase 0. Do not use the raw CHANGELOG directly.

> As an experienced technical writer and tracing expert, generate release notes for Tempo vX.Y using the following curated PR list. This list has already been sorted and approved -- include all entries.
>
> Look up each PR and provide a 2-3 sentence summary of the user-facing impact. Where appropriate, add TraceQL or configuration examples.
>
> Use `docs/sources/tempo/release-notes/v2-10.md` as a template for structure and tone.
>
> Include PR numbers and links for each entry.
>
> [Paste curated PR list here]

### Initial draft from CHANGELOG (not recommended)

Use the curated PR list prompt above when possible. Only use this prompt if you haven't completed Phase 0 curation yet, and be prepared to remove entries during review.

> As an experienced technical writer and tracing expert, generate release notes for Tempo vX.Y using the unreleased section of CHANGELOG.md.
>
> Look up each PR for additional context. Provide 2-3 sentence summaries focusing on user impact. Flag any entries that appear to be internal-only or too minor for release notes.
>
> Use `docs/sources/tempo/release-notes/v2-10.md` as a template.

### Documentation assessment for a PR list

> For each PR in the following list, assess whether it needs documentation:
>
> 1. Look up the PR description and changed files
> 2. Determine if it's user-facing (new feature, config change, behavior change, breaking change) or internal-only (refactor, test, dependency bump)
> 3. If user-facing, check: Is "Documentation added" checked in the PR checklist? Are there changes to files under `docs/`?
> 4. Search `docs/sources/tempo` for existing coverage of the feature
> 5. Classify as: docs present, docs needed, docs update needed, or no docs required
>
> Return a table with columns: PR number, title, classification, and notes (what's missing or what needs updating).
>
> [Paste PR list here]

### PR deep dive (feature evaluation)

> Access PR #XXXX. As an experienced technical writer, read the description. Based on the PR:
>
> 1. What is the user-facing impact?
> 2. Does this PR need documentation? (Is it user-facing, does it add config options, change behavior, or introduce a breaking change?)
> 3. Is "Documentation added" checked in the PR checklist? Are there doc file changes in the PR?
> 4. Is this capability already documented? (Search `docs/sources/tempo`)
> 5. If docs exist, are they complete? Do they cover the new behavior, options, and examples?
> 6. What examples would help users understand this feature?
> 7. Are there configuration options that need to be documented?

### Documentation placement analysis

> Evaluate `docs/sources/tempo/traceql/construct-traceql-queries.md` and suggest where to add an example for [FEATURE]. Include:
>
> - Why and when to use this feature
> - A practical example
> - Any version requirements or prerequisites

### Expand a specific feature

> Look up PR #XXXX. Based on the PR description and code changes:
>
> 1. Summarize the user-facing impact in 2-3 sentences
> 2. Provide a practical example (TraceQL query or configuration)
> 3. Identify any configuration options users need to set
> 4. Check if documentation was added and link to it

### Create upgrade considerations

> Review the following breaking changes and create an "Upgrade considerations" section. For each change:
>
> 1. Explain what changed and why
> 2. Describe who is affected
> 3. Provide migration steps with code examples
> 4. Note any action required before upgrading
>
> [List of breaking change PRs]

### Validate against codebase

> Validate the description of [FEATURE] in the release notes against PR #XXXX.
>
> Check that:
>
> 1. The configuration example is accurate
> 2. Any new flags or options are correctly documented
> 3. The documentation link is correct

### Final polish

> Review the release notes and:
>
> 1. Apply sentence case to all headings
> 2. Ensure all PR links are formatted as `[[PR XXXX](URL)]`
> 3. Check that documentation links use `<TEMPO_VERSION>` placeholder
> 4. Remove any CHANGELOG artifacts (usernames, category brackets)
> 5. Fix any linting issues

## Style guidelines

### Tempo-specific conventions

- Use "Grafana Tempo" on first mention, then "Tempo"
- Use "TraceQL" (not "traceql" or "Trace QL")
- Use "vParquet4", "vParquet5" (lowercase v, no space)
- Use "metrics-generator" (hyphenated)
- Reference versions as "Tempo 2.10" or "Tempo X.Y"

### PR link format

Always include PR links at the end of entries:

```markdown
[[PR 5982](https://github.com/grafana/tempo/pull/5982)]
```

For multiple PRs:

```markdown
(PRs [#5939](https://github.com/grafana/tempo/pull/5939), [#6001](https://github.com/grafana/tempo/pull/6001))
```

### Documentation links

Use the version placeholder for internal docs:

```markdown
[documentation](/docs/tempo/<TEMPO_VERSION>/path/to/doc/)
```

### TraceQL examples

Format TraceQL queries in code blocks with the `traceql` language tag:

````markdown
```traceql
{ span:childCount > 10 }
```
````

Include explanatory text after examples describing what the query does.

### Configuration examples

Use YAML code blocks for configuration:

```yaml
storage:
  trace:
    block:
      version: vParquet5
```

## Iteration checklist

After generating the initial draft, verify:

### Content completeness

- [ ] All PRs from the source list are included
- [ ] Entries are grouped logically by topic
- [ ] Every entry has a PR link
- [ ] Featured sections have practical examples
- [ ] Breaking changes are in "Upgrade considerations" with migration steps

### Documentation assessment

- [ ] Every user-facing PR classified (docs present, docs needed, docs update needed, no docs required)
- [ ] PRs with "Documentation added" checked have been verified to actually include docs
- [ ] All "docs needed" PRs have been addressed or flagged
- [ ] All "docs update needed" PRs have corresponding updates to existing pages
- [ ] Documentation gaps tracked and communicated to the team
- [ ] Every `docs needed` and `docs update needed` PR is either documented or has an explicit blocker with owner/follow-up

### Documentation coverage

- [ ] Major features have examples validated against codebase
- [ ] New intrinsics/functions are added to reference tables
- [ ] Configuration options are documented or linked
- [ ] Existing docs are updated where needed (not just release notes)

### Quality checks

- [ ] Documentation links use `<TEMPO_VERSION>` placeholder
- [ ] Headings use sentence case
- [ ] No CHANGELOG artifacts remain (usernames, brackets)
- [ ] Linter passes with no errors
- [ ] Examples prioritize practical debugging value

## Feature-by-feature workflow

For complex releases, work through major features individually across multiple sessions:

1. **Before session 1**: Complete Phase 0 (source curation). Sort the CHANGELOG manually, produce a curated PR list, and get it reviewed. This is a human-driven step.
2. **Session 1**: Generate initial draft from the curated PR list and run the documentation assessment (Phases 1-1.75)
3. **Sessions 2-N**: Deep dive on each major feature:
   - Access PR via GitHub API
   - Evaluate documentation status
   - Validate examples against codebase
   - Update existing docs if needed
4. **Final session**: Polish, validate links, run linter

This approach ensures each feature gets proper attention and documentation coverage.
