# Preparing release candidates

Dispatch `release-prep` with a version such as `3.1.0-rc.2`.
The optional `base_branch` input selects the source branch:
RCs accept `main` or the matching `release-vX.Y` branch.
With no override, RCs continue to use `main`;
final and patch releases use the matching release branch.

For RC2 from an existing release branch:

1. Land the workflow update on `main` and on `release-v3.1`.
   The merged prep PR runs the release branch's copy of `release-tag`.
2. Dispatch `release-prep` with `version=3.1.0-rc.2`,
   `base_branch=release-v3.1`, and `validate_only=true`.
3. After validation succeeds, repeat with `validate_only=false`
   to open the bot-authored prep PR against `release-v3.1`.
4. Review its commit, checks, `VERSION`, and changelog before merging.
   Merging publishes the tag, binaries, images, and GitHub prerelease.

When the exact release section already exists in `CHANGELOG.md`
and no pending entries remain, preparation preserves the changelog
and updates `VERSION`.
An existing section with pending entries, a duplicate heading,
or an empty prepared section fails validation.
This prevents preparation from generating the same release section twice.

RC preparation skips image-tag bumps and publishes as a prerelease.
Bot authorship, the `release-prep` label, the prep branch name,
and the matching base branch remain required for automatic tagging.

Run the workflow shell regression checks with
`python3 tools/release-workflows_test.py`.
