---
name: changelog-entry
description: Create a Tempo CHANGELOG.md entry for the current PR using the required tag set, ordering, and concise line format
allowed-tools: Bash Read Grep Edit
---

# Tempo changelog entry

Use this skill whenever deciding whether to add, or when updating, a `CHANGELOG.md` entry for a Tempo change.

## Hard rules

1. Add changelog entries only for user-facing or release-noteworthy Tempo changes under `## main / unreleased`.
2. Do not add changelog entries for docs-only changes.
3. Use only these tags, in this order:
   1. `[CHANGE]`
   2. `[FEATURE]`
   3. `[ENHANCEMENT]`
   4. `[BUGFIX]`
4. Do not invent tags. In particular, do not use `[SECURITY]`, `[PERFORMANCE]`, `[DOCS]`, `[DEPRECATION]`, or `[BREAKING]`.
5. Every changelog entry is exactly one bullet line. Do not add wrapped continuation lines.
6. Keep the description concise: one user-facing sentence, target 120–180 characters, hard maximum 220 characters before the PR link.
7. Do not include implementation detail unless it is the user-visible thing being changed.
8. Do not create subsection headings inside `main / unreleased`.

If no entry is required, do not edit `CHANGELOG.md`; report `No changelog entry needed` with the reason.

## Exact line format

Use this format:

```markdown
* [TAG] area: concise user-facing description. [#PR](https://github.com/grafana/tempo/pull/PR) (@github-login)
```

Examples:

```markdown
* [CHANGE] query-frontend: replace `query_shards` with `blocks_per_shard` for trace lookup sharding. [#7105](https://github.com/grafana/tempo/pull/7105) (@mdisibio)
* [FEATURE] TraceQL metrics: support arithmetic operations. [#6866](https://github.com/grafana/tempo/pull/6866) (@ruslan-mikhailov)
* [ENHANCEMENT] cache: enforce a configurable max item size for Redis. [#7311](https://github.com/grafana/tempo/pull/7311) (@electron0zero)
* [BUGFIX] livestore: avoid cache collisions between instant and range metrics queries. [#7290](https://github.com/grafana/tempo/pull/7290) (@ruslan-mikhailov)
```

### Format details

- Start with `* [`.
- Use the singular tag `[CHANGE]`, not `[CHANGES]`.
- Prefer a lowercase component scope followed by a colon, for example `querier:`, `tempodb:`, `jsonnet:`, `tempo-cli:`.
- If there is no clear component, omit the scope but keep the rest of the format.
- End the description with a period before the PR link.
- Include exactly one PR link and exactly one GitHub author handle.
- If the PR has multiple authors, use the PR author's GitHub login from `gh pr view`. Do not list reviewers or co-authors.
- For breaking changes, use `[CHANGE]` and start the description with `**BREAKING CHANGE**` only when the PR actually requires user action.

## Get PR metadata with `gh`

If the user gives a PR number:

```bash
gh pr view PR_NUMBER --repo grafana/tempo --json number,url,author,title,body,labels,files
```

If working from the current branch:

```bash
gh pr view --repo grafana/tempo --json number,url,author,title,body,labels,files
```

Extract:

- PR number: `.number`
- PR link: `.url`
- GitHub author: `.author.login`
- Whether a changelog entry is required: title, body, labels, and changed files
- User-visible behavior to summarize when an entry is required

If `gh pr view` cannot identify a PR, ask the user for the PR number. Do not fabricate the PR number, link, or author.

## Choose the tag

Use the first matching category below:

1. `[CHANGE]`
   - Breaking change, removed behavior, changed default, migration step, deprecation with user action, or other user-visible behavior change.
2. `[FEATURE]`
   - New user-facing capability, API, command, query feature, config-driven mode, or major workflow.
3. `[ENHANCEMENT]`
   - Improvement to existing behavior, observability, performance, config surface, dashboard, alert, or operational workflow.
4. `[BUGFIX]`
   - Fix for incorrect behavior, regression, panic, data loss, leak, race, bad validation, or wrong metric/query result.

Security fixes are normally `[BUGFIX]`. Security hardening that is not a bug fix is normally `[ENHANCEMENT]`. Never use `[SECURITY]`.

## Insert in the right place

1. Read only the `## main / unreleased` section of `CHANGELOG.md`.
2. Keep entries grouped in this exact order:
   - all `[CHANGE]` entries
   - all `[FEATURE]` entries
   - all `[ENHANCEMENT]` entries
   - all `[BUGFIX]` entries
3. Insert the new line at the end of its tag group.
4. If the tag group does not exist yet, insert it after the previous allowed tag group, or before the next allowed tag group.
5. Preserve relative order within each existing tag group unless the user explicitly asks to sort or clean up the changelog.
6. If the target section already contains unknown tags or out-of-order entries, report the drift and still insert the new entry in the canonical location. Do not introduce more drift.

## Quality checklist before editing

Before modifying `CHANGELOG.md`, verify the proposed line:

- The PR needs a changelog entry; it is not docs-only, tests-only, CI-only, or an internal-only refactor.
- Tag is one of `[CHANGE]`, `[FEATURE]`, `[ENHANCEMENT]`, `[BUGFIX]`.
- The line is a single bullet and matches the exact format.
- The description is concise and user-facing.
- PR number, link, and author came from `gh pr view` or were provided by the user.
- The insertion point preserves the required tag order.

## Final response

If an entry was added, return only:

1. The changelog line added.
2. The insertion location, for example `CHANGELOG.md: main / unreleased -> [ENHANCEMENT]`.
3. Any drift noticed, such as unknown tags or existing out-of-order entries.

If no entry was needed, return only:

`No changelog entry needed: <reason>`
