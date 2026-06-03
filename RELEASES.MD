# Release Candidates

- Create a branch named `changelog-<version>-<rc.#>`. Generate the changelog section from the
  `.chloggen/` entries by running `make chlog-update VERSION=v<version>-<rc.#>` (this collates the
  entries into `CHANGELOG.md` under the version heading and deletes the consumed files). Review the
  result, then merge to main.
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
- Submit a PR updating the changelog for the full version. Because the RC above already ran
  `make chlog-update` (consuming the `.chloggen/` entries), this is usually just renaming the
  `# v<version>-<rc.#>` heading in `CHANGELOG.md` to the final `# v<version>`. If new entries landed
  since the RC, run `make chlog-update VERSION=v<version>` instead to collate them.
- In [github releases](https://github.com/grafana/tempo/releases) there should be a "Draft" release.
  Pretty up the changelog, add it to the release notes and hit "Publish release". Make sure that
  "Set as the latest release" is checked.
- Update helm
  - Submit PRs to `github.com/grafana/helm-charts` to update to the newly cut version.
    - One PR each for the `tempo`, `tempo-distributed` and `tempo-vulture` helm charts.
    - Search the chart for the previous version and udpate each version number to current.

# Patch Releases

Patches release should be cut for serious bug fixes or security issues.

- A release branch should exist per instructions above. It will be named:
  `git checkout release-vX.Y`
- Commits should already have been backported to this branch using backport automation, but if 
  you need to add more you can cherry-pick them:
   `git cherry-pick <commit hash>`
- Generate the patch version's changelog section on the release branch by running
  `make chlog-update VERSION=v1.0.1` (this collates the backported `.chloggen/*.yaml` entries into
  `CHANGELOG.md` under the new version heading and deletes the consumed files), then commit it.
- Push the commit  
  `git push`
- Create and push a tag to initiate the release. This will be pushed to the release branch.
  `git tag -a v1.0.1`  
  `git push origin v1.0.1`  
  This triggers the build.
- Submit a PR to main adding the new patch version's section to `CHANGELOG.md` so the released notes
  are reflected there too.
- In [github releases](https://github.com/grafana/tempo/releases) there should be a "Draft" release.
  Pretty up the changelog, add it to the release notes and hit "Publish release". Make sure that
  "Set as the latest release" is checked.
- Submit a PR to the Helm repo as described above.
