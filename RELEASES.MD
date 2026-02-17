# Release Candidates

- Create a branch named `changelog-<version>-<rc.#>`. Clean up the changelog, add the version heading
  and merge to main. 
- Push a semver tag to main on the merge commit above.  Something like:
  - `git tag -a v1.2.0-rc.0`
  - `git push origin v1.2.0-rc.0`
- This will initiate the build process in Github Actions.  The tagged docker image should
  be available here shortly: https://hub.docker.com/r/grafana/tempo/tags?page=1&ordering=last_updated
- A Github Release Draft should also be available here:  https://github.com/grafana/tempo/releases
  - Copy over the CHANGELOG entries for the release
  - Call out contributors for their work
  - Cull unnecessary changes that don't impact the Tempo binary or deployment
  - Call out breaking changes!
- Publish the release making sure that "This is a pre-release" is checked.

# Releases

This document details release procedures for Tempo.

- Create a release branch. This may or may not be on the same commit as the release candidate above.
  - Name the branch like `release-v2.2`
- Follow all steps in [Release Candidates](#release-candidates) except:
  - Drop the `-rc.#` postfix from the tag. For instance use `v1.2.0` instead. Something like:
    - `git tag -a v1.2.0`
    - `git push origin v1.2.0`
  - Make sure that the "This is a pre-release" is unchecked when publishing the release.
- Submit a PR cleaning up the changelog and moving everything under "main/unreleased" to be under
  the newly minted version.
  - Given that the changelog was already cleaned up for the RC above. This will likely be simply
    renaming the release candidate to the full version.
- In [github releases](https://github.com/grafana/tempo/releases) there should be a "Draft" release.
  Pretty up the changelog, add it to the release notes and hit "Publish release". Make sure that
  "Set as the latest release" is checked.
- Update helm
  - Submit PRs to `github.com/grafana/helm-charts` to update to the newly cut version.
    - One PR each for the `tempo`, `tempo-distributed` and `tempo-vulture` helm charts.
    - Search the chart for the previous version and udpate each version number to current.

# Patch Releases

Patch releases should be cut for serious bug fixes or security issues.

## 1. Check out the release branch
A release branch should already exist from the minor release. Check it out:
  `git checkout release-vX.Y`

## 2. Backport fixes
Backport fixes to the release branch. This is currently a manual process.
You need to cherry-pick each commit and open a PR to be merged into the release branch.
  `git cherry-pick <commit hash>`

## 3. Update the changelog
- Verify all backported fixes have entries in the CHANGELOG. Cross-reference the commits on the release branch since the last tag (`git log --oneline $(git describe --tags --abbrev=0)..release-vX.Y`) against the CHANGELOG entries.
- Add any missing entries and update the heading from `main / unreleased` to the new patch version (e.g. `v2.10.1`).
- Open a PR with the changelog update to the release branch.

## 4. Create and push the tag
Tag the release branch and push. This triggers the `release` and `docker` workflows in GitHub Actions.
  `git tag -a vX.Y.Z -m "vX.Y.Z"`
  `git push origin vX.Y.Z`

## 5. Verify CI
Check that all workflows triggered by the tag succeeded:
- `release` — creates the GitHub Release draft and binaries
- `docker` — builds per-arch images and creates multi-arch manifests
- `publish-technical-documentation-release` — syncs docs (may show as failed if docs are already up to date; this is harmless)

## 6. Publish the GitHub Release
In [GitHub Releases](https://github.com/grafana/tempo/releases) there should be a "Draft" release. Add the changelog entries to the release notes. Make sure "Set as the latest release" is checked and hit "Publish release".

## 7. Submit a changelog PR to main
Submit a PR to `main` moving the relevant entries from "main/unreleased" to be under the newly minted version.
