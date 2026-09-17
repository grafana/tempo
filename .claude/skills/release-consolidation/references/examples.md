# Consolidation examples

These examples are synthetic, not claims about a Tempo release.
PR numbers, contributor handles, and vulnerability identifiers are placeholders.

## Dependency upgrades with security history

Inputs:

- PR A upgrades a dependency to version B and fixes CVE X.
- PR C upgrades it again to version D and fixes CVE Y.
- PR E aligns a build image with version D without identifying a security fix.

Verify the release's dependency manifests and build inputs.
Describe version D as the final state.
Keep CVE X associated with PR A and CVE Y with PR C;
list both references in the Security entry if their consolidation is justified.
Do not describe PR E as fixing a vulnerability without evidence.
Keep it under Changes if it has an independent user impact.

A later version number alone is not proof that CVE X remains fixed.
If the evidence is unavailable, flag the claim rather than deleting or expanding it.

## Feature development and a superseded warning

Inputs:

- PR A introduces an experimental API skeleton;
  its note says it always returns an empty result.
- PR B implements the API.
- PR C fixes a boundary condition in that implementation.

Check the implementation and tests at the release SHA.
If the API now returns results, write one final-state description
and retain A, B, and C as references.
Remove the stale empty-result claim, not the experimental designation,
unless code or reviewed documentation also supports that change.
Retain any opt-in requirement or configured limit.

Do not fold an unrelated fix merely because it belongs to the same component.

## Contributor credit is a constraint

Inputs:

- PR A adds a capability and credits `contributor-a`.
- PR B improves it and credits `contributor-b`.

The current entry schema has one `user` field.
Do not create a comma-separated pseudo-handle or silently choose one author.
Keep separate entries unless an approved representation preserves both credits
and their contribution associations in the generated Markdown.
Do not alter the generator in the release PR to enable the fold.

## One PR, two release-note categories

One entry describes both a new default and an independent bug fix.
The default change requires operator action during a mixed-version rollout.

Propose separate Breaking changes and Bug fixes entries,
both retaining the original PR reference and applicable credit.
Keep the exact option name, default, and rollout guidance in the breaking entry.
Repeated references are valid;
deduplication is about repeated claims, not forcing one bullet per PR.

## Existing release-prep PR

The PR already contains a generated version section, a `VERSION` change,
and deletion of normal and `_migrated_*` YAML entries.

Recover the pre-generation inputs from Git in a scratch worktree.
Apply approved source edits there and generate once.
Transfer only the resulting changelog-section change to the prep branch.
Confirm that consumed entries remain deleted in the PR diff,
its `VERSION` change remains intact,
and prior release history has not changed.

Do not restore YAML on the prep branch and leave it pending.
Do not rerun generation on the already-generated version section.
Do not include a generator fix in this PR.

## Final release after multiple RCs

RC preparation already consumed entries A and B.
Only entry C remains pending for the final release.

A final section based on C alone would omit A and B.
Recover the RC source sets and combine them with C for the approved final section.
Preserve published RC tags and release records.
If the sources or safe regeneration path cannot be established,
stop and ask for a maintainer-approved reconstruction.

## Evaluating the skill

[The evaluation cases](../evals/evals.json) cover planning and approved application.
Use the repository's [evaluation procedure](../../evals/README.md),
but construct the synthetic fixtures described in each case
instead of selecting unrelated live PRs.
Run application cases only in a disposable clone or worktree
with no credentials that can publish a release.
Mock PR evidence where specified and record the mock data.

Inspect both the response and the resulting diff.
Checking JSON syntax and Markdown links does not execute these behavioral evaluations.
Record which assertions were exercised;
do not report unrun cases as passing.
