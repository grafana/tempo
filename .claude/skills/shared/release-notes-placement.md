# Release notes placement

Read this during Phase 0 sorting.
Assign every changelog entry one label.

## Labels

| Label | What to do |
|-------|------------|
| **Highlight** | New for this release. Intro bullet and featured section, with examples. |
| **Brief include** | User-facing, not headline. Short bullet under Features, Upgrade, or Bug fixes. |
| **Cite prior notes** | Already covered in an earlier X.Y. Mention it and link there. Do not rewrite. |
| **Exclude** | Changelog only. |
| **Uncertain** | Can't tell from the diff. Hold for human review. Do not exclude. |

**Fold** is not a placement label.
Several PRs that are the same change get one description and a list of PR numbers.
Follow-up fixes cite the parent feature, not each fix.

## Which pattern

Default to a normal minor: most entries appear.

When this release graduates work that earlier notes already documented,
cite those notes.
Keep this file for what changed now:
now default or GA, breaking changes, and net-new items.

Tempo 2.10 is a normal minor.
Tempo 3.0 cited 2.9 and 2.10 for architecture work already documented there.

## Always keep

Breaking changes, Go upgrades, new config or flags, deprecations,
and bugs users could have hit.

## Exclude

No user-facing effect once you check the diff, not just the label:
tests and harnesses, docs-only, mixins and dashboards, vendor bumps,
internal refactors, example-config-only, internal metrics plumbing.

Can't tell from the diff? Label it uncertain, not exclude.

## Sanity check

On a normal minor, most chloggen entries should appear.
If almost everything is excluded, revisit.
Do not use a major-version cite-back pass as the omit rate for a minor.
