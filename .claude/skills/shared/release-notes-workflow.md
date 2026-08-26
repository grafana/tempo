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
- Source material: a curated PR list from the engineering team,
  pending entries in `.chloggen/*.yaml` (or `make chlog-preview` output),
  or a collated `# vX.Y.Z` section in `CHANGELOG.md` after `make chlog-update`

Unreleased work never lives in `CHANGELOG.md`.
See [`.chloggen/README.md`](../../../.chloggen/README.md)
and the [`changelog-entry`](../../../.claude/skills/changelog-entry/SKILL.md) skill
for how individual PR entries are created;
this workflow only consumes those entries to draft release notes.

## Workflow phases

### Phase 0: Source curation (human-driven, AI-assisted)

The pending changelog entries must be sorted and grouped before drafting begins. An AI assistant can do the heavy lifting -- reading each PR, understanding context, and proposing groupings -- but a human must review, adjust, and approve the final result. Do not proceed to Phase 1 until this is complete.

The goal of this phase is to understand the shape of the release: what the major feature areas are, which changelog entries belong to each area, and what to include or exclude from the release notes.

#### Step 1: Collect pending changelog entries

Get the input set for the upcoming release:

- **Before a release is collated**: pending entries in `.chloggen/*.yaml`
  (ignore `TEMPLATE.yaml` and `config.yaml`).
  Prefer `make chlog-preview` when `issues` are filled in for every entry.
  If preview fails because some entry has `issues: []`,
  parse the YAML directly and resolve PR numbers
  from git history, the branch, or GitHub.
- **After `make chlog-update VERSION=vX.Y.Z`**:
  use the versioned `# vX.Y.Z` section in `CHANGELOG.md`.

CI already filters this input set:
PRs labeled `Skip Changelog` or `dependencies`,
labeled `type/docs`, or with a `chore:`-prefixed title
typically don't get a `.chloggen` entry at all,
so the pending entries are already a filtered starting point.

#### Step 2: Identify known headline features

Before sorting the pending entries, ask: **are the main features of this release already known?**

The engineering team, product manager, or writer may already know which features define this release. If so, these known headline features and their associated PRs become the primary emphasis of the release notes. Everything else in the changelog entries gets organized around them.

If the main features are known:
- List them along with the key PRs associated with each feature
- These features become the top-level feature areas for sorting
- The AI-assisted sorting in Step 3 should group remaining changelog entries around these known areas and identify any additional areas the team may not have mentioned

If the main features aren't known yet:
- Proceed to Step 3 and let the AI propose the major feature areas by reading through the changelog entries
- Review the AI's proposed areas with the engineering team in Step 4 to confirm

#### Step 3: AI-assisted sorting

Provide the pending changelog entries (from `.chloggen` or `chlog-preview`) to the AI assistant, along with the known headline features if available. Ask it to:

1. **Look up each PR** to read the description, understand the change, and assess user impact.

2. **Identify or confirm the major feature areas** for this release:
   - If headline features were provided in Step 2, use those as the primary feature areas and group associated PRs under them. Then identify any additional areas from the remaining entries.
   - If no headline features were provided, identify the major feature areas from the changelog entries themselves. These aren't fixed categories -- they emerge from the entries. The AI should read through all entries and propose the themes that define the release.

   For example, the Tempo 2.10 release had these major feature areas:
   - TraceQL (new query capabilities like `= nil`, `span:childCount`, `minInt`/`maxInt`)
   - Metrics-generator (entity-based limiting, overflow series, cardinality estimation)
   - vParquet5 (production readiness, dedicated columns, blob detection)
   - LLM-optimized API responses
   - User-configurable overrides
   - Project Rhythm (new architecture, experimental)

   A different release might have entirely different major areas, for example, query performance, multi-tenancy, or new storage backends. Let the changelog entries tell you what the release is about.

   **`change_type` and `component` from `.chloggen` are hints, not the release-notes grouping.** Use `change_type` as a first pass: `breaking` entries are candidates for Upgrade considerations, `bug_fix` for Bug fixes, `security` for Security fixes. `component` (for example, `metrics-generator`, `traceql`) hints at feature area but doesn't replace it -- a single feature area often spans several components, and a single component often spans several feature areas.

   `make chlog-preview` already sorts entries by `component` within each `change_type` section --
   component-level grouping is already free.
   What this step actually adds is the cross-cutting *feature-area* grouping,
   not re-deriving component buckets by hand.

3. **Group every changelog entry** into the feature areas:
   - Assign each entry to the feature area where it has the most user impact. PRs associated with known headline features should be grouped under those areas first.
   - Within each group, weight the entries: identify which are headline items versus supporting changes, minor improvements, or bug fixes.
   - Assign every entry a placement label using [release-notes-placement.md](release-notes-placement.md).
   - Use **Uncertain** when you can't decide -- these need human judgment.

4. **Return a structured, grouped list** organized by feature area, with entries ordered by importance within each group and exclusions listed separately. Known headline features should appear first.

#### Step 3b: Place each entry

Read [release-notes-placement.md](release-notes-placement.md)
and assign every entry a label: highlight, brief include, cite prior notes, exclude, or uncertain.
Fold sibling PRs into one description as that file describes --
folding is how an include is represented, not a skip.

Docs-only and dependency-only PRs increasingly skip `.chloggen` entirely
(see the skip labels in Step 1),
so the share of surviving entries that appear in the notes is often high.
Treat that as a sanity check for a normal minor, not a hard target.

#### Step 4: Human review

The AI's proposed grouping is a starting point, not the final answer. Review the grouped list and:

- Adjust feature area names and boundaries (the AI may split or merge areas incorrectly)
- Move entries between groups if the AI misjudged the primary feature area
- Override placement labels where the AI lacked context
- Resolve uncertain entries
- Confirm which feature areas deserve featured sections versus brief mentions

#### Step 5: Coordinate with the engineering team

Review the grouped list with the engineering team to:
- Confirm the major feature areas are correct and complete
- Identify the top 3-5 features for the introduction highlights
- Identify breaking changes requiring upgrade documentation
- Flag features that need documentation updates
- Resolve any remaining entries you're unsure about

#### Step 6: Produce the curated input, then stop

The final output of Phase 0 is a curated, grouped PR list organized by major feature area, with entries weighted by importance within each group. This list is the input for all subsequent phases.

**STOP here.** Output the grouped list in a form the writer can edit -- highlight and brief include (by feature area, weighted), cite prior notes (with the earlier X.Y to link), fold (with fold-into target), exclude (with reason), uncertain (with why) -- and end the turn. Do not start Phase 1, 1.5, 1.75, or drafting in the same turn, even for a short source list. Resume only after the writer explicitly approves (for example, "approved") or sends a revised list; a revision gets a new grouped list and stops again, the same way.

This also applies to the "Initial draft from pending entries (not recommended)" example prompt below -- it's a shortcut for skipping curation, not for skipping this approval. And to the [feature-by-feature workflow](#feature-by-feature-workflow) at the end of this doc: session 1 may run Phases 1 through 1.75 only when Phase 0 was already approved in an earlier turn.

Once approved, work through Phases 1 to 5 one feature area at a time using the [feature-by-feature workflow](#feature-by-feature-workflow) -- the default approach for any release with a nontrivial number of entries, not only unusually complex ones.

### Phase 1: Input gathering

Using the curated PR list from Phase 0, gather what's needed to draft each entry: user impact, config or flag names, migration steps. This is drafting context only -- documentation-gap classification happens in Phase 1.5, not here. (Phase 1.5's `docs-pr-check` already does its own PR lookup and checks for changed `docs/` files; duplicating that here just redoes the same work with a shallower schema.)

Most entries don't need a fresh PR lookup: `.chloggen` notes are already written for release-notes consumption (per `.chloggen/README.md`, the note opens with user impact). Draft directly from the note, and look up the PR only when:

- the entry is a headline, breaking, or security item that needs a deeper writeup than the note provides,
- Phase 0 flagged the entry as uncertain,
- a specific config or behavior claim needs verification beyond what Phase 4's code checks can confirm on their own.

When you do look up a PR, read the description and linked issues for context the note doesn't cover, and confirm configuration changes, new flags, or migration steps.

### Phase 1.5: Documentation assessment

Run `/docs-pr-check` on every Phase 0 **highlight**, **brief include**, **cite prior notes**, and **fold** entry. Placement is a release-notes-narrative decision, not a documentation-completeness decision -- a folded or cited PR can still be an undocumented config change, and this phase exists specifically to catch that. Skip Phase 1.5 only for entries Phase 0 **excluded** for a reason already synonymous with "no docs needed": docs-only, dependency/vendor bump, pure internal refactor, test/validation infrastructure, internal metrics plumbing ([release-notes-placement.md](release-notes-placement.md) is the source list). For any other exclude reason, still run it.

This is a critical review step that prevents features from shipping without adequate docs.

See [`.claude/skills/docs-pr-check/SKILL.md`](../../../.claude/skills/docs-pr-check/SKILL.md) for the full classification process, criteria, and return format.

### Phase 1.75: Documentation gap resolution

For each PR classified as "docs needed" or "docs update needed", create or update the required documentation pages.

**Present the Phase 1.5 gap list and confirm before running `/docs-pr-write`.** Unlike Phase 1.5's classification, this phase writes to real documentation pages. Do not run it automatically the moment Phase 1.5 finishes -- there may be reasons to defer or rescope it (batching with other doc work, waiting on engineering input for an ambiguous gap, running it only for a subset of entries first). Resume once the writer confirms whether to proceed, and with what scope.

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
4. Run `.claude/skills/shared/vale-compact.sh <file>` to fix style issues (a compact wrapper over `vale` -- same findings, without its per-finding boilerplate)
5. Apply sentence case to all headings
6. Remove chloggen formatting artifacts: `component:` prefixes, emoji section headings (🚀, 💡, 🧰, and so on), and `(@handle)` suffixes. Older, already-shipped CHANGELOG sections may still use bracketed `[CHANGE]`-style tags instead -- leave those as historical text; don't retrofit them.

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

### Changelog curation assist (Phase 0)

Use these prompts to help with the sorting process, but **a human must review and approve the final grouped list** before proceeding.

#### Sort pending entries with known headline features

Use this prompt when the main features of the release are already known. The known features become the primary emphasis, and the remaining entries are organized around them.

> Here are the pending changelog entries for Tempo vX.Y (from `.chloggen/*.yaml` or `make chlog-preview` output). The main features of this release are already known:
>
> - [Feature 1]: [brief description, key PRs if known]
> - [Feature 2]: [brief description, key PRs if known]
> - [Feature 3]: [brief description, key PRs if known]
>
> Sort the entries using these headline features as the primary feature areas. For each entry:
>
> 1. Look up the PR to read the description and understand the change
> 2. Assess the user impact
>
> Then:
>
> 1. **Group entries under the known headline features first.** PRs associated with each headline feature should be grouped together, with the key PRs as the headline items.
> 2. **Identify any additional feature areas** from the remaining entries that don't fit under the known features.
> 3. **Order entries within each group** by importance: headline items first, then supporting changes, then minor fixes.
> 4. **Assign a placement label** using [release-notes-placement.md](release-notes-placement.md): highlight, brief include, cite prior notes, exclude, or uncertain. Fold sibling PRs into one description. List exclusions separately with a brief reason.
> 5. **For uncertain entries**, explain why you couldn't decide. These need human judgment.
>
> Return the result as a structured, grouped list with the known headline features listed first. I'll review and adjust your groupings before we proceed.
>
> [Paste chlog-preview output or the .chloggen entries here]

#### Sort pending entries without known headline features

Use this prompt when the main features aren't known yet. The AI proposes the major feature areas by reading through the pending entries.

> Here are the pending changelog entries for Tempo vX.Y (from `.chloggen/*.yaml` or `make chlog-preview` output). I need you to sort these entries into major feature areas for the release notes.
>
> For each entry:
>
> 1. Look up the PR to read the description and understand the change
> 2. Assess the user impact (is this a headline feature, a supporting enhancement, a minor fix, or an internal-only change?)
>
> Then:
>
> 1. **Identify the major feature areas** for this release. These should be the themes that emerge from the entries themselves (for example, "TraceQL enhancements", "metrics-generator cardinality management", "vParquet5 production readiness"). Don't use fixed categories; let the entries tell you what this release is about.
> 2. **Group every entry** into the proposed feature areas. Assign each entry to the area where it has the most user impact.
> 3. **Order entries within each group** by importance: headline items first, then supporting changes, then minor fixes.
> 4. **Assign a placement label** using [release-notes-placement.md](release-notes-placement.md): highlight, brief include, cite prior notes, exclude, or uncertain. Fold sibling PRs into one description. List exclusions separately with a brief reason.
> 5. **For uncertain entries**, explain why you couldn't decide. These need human judgment.
>
> Return the result as a structured, grouped list organized by feature area. I'll review and adjust your groupings before we proceed.
>
> [Paste chlog-preview output or the .chloggen entries here]

#### Adjust groupings after human review

Use this prompt after reviewing the AI's initial sort, if you need to refine the groupings.

> I've reviewed your proposed feature area groupings. Here are my adjustments:
>
> [Describe adjustments: moved entries, renamed areas, changed placement labels, etc.]
>
> Please update the grouped list with these changes and return the revised version.

### Initial draft from curated PR list

This prompt requires a curated PR list from Phase 0. Do not use the pending `.chloggen` entries or `chlog-preview` output directly.

> As an experienced technical writer and tracing expert, generate release notes for Tempo vX.Y using the following curated PR list. This list has already been sorted and approved -- include all entries.
>
> Look up each PR and provide a 2-3 sentence summary of the user-facing impact. Where appropriate, add TraceQL or configuration examples.
>
> Use `docs/sources/tempo/release-notes/v2-10.md` as a template for structure and tone.
>
> Include PR numbers and links for each entry.
>
> [Paste curated PR list here]

### Initial draft from pending entries (not recommended)

Use the curated PR list prompt above when possible. Only use this prompt if you haven't completed Phase 0 curation yet, and be prepared to remove entries during review.

> As an experienced technical writer and tracing expert, generate release notes for Tempo vX.Y using the pending changelog entries in `.chloggen/*.yaml` (or `make chlog-preview` output).
>
> Look up each PR for additional context. Provide 2-3 sentence summaries focusing on user impact. Flag any entries that appear to be internal-only or too minor for release notes.
>
> Use `docs/sources/tempo/release-notes/v2-10.md` as a template.

### Documentation assessment for a PR list

> Run `/docs-pr-check` on the following PRs and give me the classification table and gap summary:
>
> [Paste PR list here]

This is Phase 1.5 -- don't restate `docs-pr-check`'s classification steps here; the skill owns that logic (see [`.claude/skills/docs-pr-check/SKILL.md`](../../../.claude/skills/docs-pr-check/SKILL.md)). Keeping both in sync by hand invites drift.

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
> 4. Remove any chloggen artifacts (`component:` prefixes, emoji section headings, `(@handle)` suffixes)
> 5. Run `.claude/skills/shared/vale-compact.sh` and fix any linting issues

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
- [ ] No chloggen artifacts remain (`component:` prefixes, emoji headings, `(@handle)` suffixes)
- [ ] Linter passes with no errors
- [ ] Examples prioritize practical debugging value

## Feature-by-feature workflow

The default approach for any release with a nontrivial number of entries, not only unusually complex ones: work through major features individually across multiple sessions, rather than drafting everything in one pass.

1. **Before session 1**: Complete Phase 0 (source curation) and get the grouped list explicitly approved -- the Step 6 stop applies here too. Session 1 may run Phases 1 through 1.75 only once that approval exists from a prior turn, not as a way to fold curation and drafting into the same session.
2. **Session 1**: Generate initial draft from the curated PR list and run the documentation assessment (Phases 1-1.75)
3. **Sessions 2-N**: Deep dive on each major feature:
   - Access PR via GitHub API
   - Evaluate documentation status
   - Validate examples against codebase
   - Update existing docs if needed
4. **Final session**: Polish, validate links, run `.claude/skills/shared/vale-compact.sh`

This approach ensures each feature gets proper attention and documentation coverage.
