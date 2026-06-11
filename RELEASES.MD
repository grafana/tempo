# Release Candidates

- Create a branch named `changelog-<version>-<rc.#>`. Generate the changelog section from the
  `.chloggen/` entries by running `make chlog-update VERSION=v<version>-<rc.#>` (this collates the
  entries into `CHANGELOG.md` under the version heading and deletes the consumed files). Review the
  result, then merge to main.
- In the same branch, set the `VERSION` file to `<version>-<rc.#>` (no leading `v`). The file is the
  source of truth for the version baked into binaries; the `docker` and `release` workflows refuse
  to build a tag that does not match it. The `release-prep` workflow does both of these steps
  automatically.
- Push a semver tag to main on the merge commit above.  Something like:
  - `git tag -a v1.2.0-rc.0`
  - `git push origin v1.2.0-rc.0`
- This will initiate the build process in Github Actions. The `docker` workflow publishes
  tagged images to GAR under `us-docker.pkg.dev/grafanalabs-global/dockerhub-tempo-prod-mirror/`.
  `gar-image-mirror` asynchronously mirrors those images to Docker Hub; they should be
  available here shortly after the workflow succeeds: https://hub.docker.com/r/grafana/tempo/tags?page=1&ordering=last_updated
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
- Submit a PR to `main` bumping the `VERSION` file to the next minor's dev version (e.g. `2.3.0-dev`),
  so main builds report the version under development rather than the released one.
- Follow all steps in [Release Candidates](#release-candidates) except:
  - Set the `VERSION` file to the full `<version>` (drop the `-rc.#`) on the release branch.
  - Drop the `-rc.#` postfix from the tag. For instance use `v1.2.0` instead. Something like:
    - `git tag -a v1.2.0`
    - `git push origin v1.2.0`
  - Make sure that the "This is a pre-release" is unchecked when publishing the release.
- Submit a PR updating the changelog for the full version. Because the RC above already ran
  `make chlog-update` (consuming the `.chloggen/` entries), this is usually just renaming the
  `# v<version>-<rc.#>` heading in `CHANGELOG.md` to the final `# v<version>`. If new entries landed
  since the RC, run `make chlog-update VERSION=v<version>` instead to collate them.
- Update the tempo image tag in example configs and operations files, then regenerate compiled jsonnet.
  Run `make bump-tempo-image-tag TEMPO_IMAGE_TAG=X.Y.Z && make jsonnet` and open a PR to the release
  branch. This is intentionally done before tagging so that users referencing the examples or docs
  get the correct released version rather than `latest` or a stale one.
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
- Verify all backported fixes have a `.chloggen/` entry on the release branch. Cross-reference the commits since the last tag (`git log --oneline $(git describe --tags --abbrev=0)..release-vX.Y`) against the entries in `.chloggen/`.
- Add any missing entries with `make chlog-new`.
- Generate the release section: `make chlog-update VERSION=vX.Y.Z` (e.g. `v2.10.1`). This collates all `.chloggen/*.yaml` files into `CHANGELOG.md` under the new version heading and deletes the consumed entry files.
- Set the `VERSION` file to `X.Y.Z` (no leading `v`). The `docker` and `release` workflows refuse to build a tag that does not match it. The `release-prep` workflow does this automatically.
- Open a PR with the changelog update to the release branch.

## 4. Update the tempo image tag

> **Note:** This step is intentionally performed before tagging and the `docker` CI workflow.
> Example configs and operations files must reference the released version so that users who follow
> the documentation or deploy from those files get the correct image, not `latest` or a stale version.

Run the following from the release branch:
```
make bump-tempo-image-tag TEMPO_IMAGE_TAG=X.Y.Z
make jsonnet
```

Review the changes with `git diff`, then open a PR to the release branch and merge it before proceeding.

## 5. Create and push the tag
Tag the release branch and push. This triggers the `release` and `docker` workflows in GitHub Actions.
  `git tag -a vX.Y.Z -m "vX.Y.Z"`
  `git push origin vX.Y.Z`

## 6. Verify CI
Check that all workflows triggered by the tag succeeded:
- `release` — creates the GitHub Release draft and binaries
- `docker` — builds per-arch images and creates multi-arch manifests in GAR under `us-docker.pkg.dev/grafanalabs-global/dockerhub-tempo-prod-mirror/` for asynchronous mirroring to Docker Hub
- Verify the tag appears in GAR first, then on Docker Hub after the mirror catches up
- `publish-technical-documentation-release` — syncs docs (may show as failed if docs are already up to date; this is harmless)

## 7. Publish the GitHub Release
In [GitHub Releases](https://github.com/grafana/tempo/releases) there should be a "Draft" release. Add the changelog entries to the release notes. Make sure "Set as the latest release" is checked and hit "Publish release".

## 8. Submit a changelog PR to main
Submit a PR to `main` adding the new version's section to `CHANGELOG.md` so the released notes are reflected on `main`. Either cherry-pick the changelog commit from the release branch (step 3), or run `make chlog-update VERSION=vX.Y.Z` against the corresponding `.chloggen/` entries on `main`.

# Automated Releases (experimental)

> **Status:** new and not yet verified end to end. The manual procedures above remain the
> source of truth until this flow has been proven on a real release. Both can coexist.

This flow drives release candidates, minor releases, and patch releases with two GitHub
Actions workflows that reuse the existing `release.yml` and `docker.yml` build pipelines:

- **`release-prep`** (`workflow_dispatch`) — prepares the changelog, sets the `VERSION`
  file, and (for non-RC releases) bumps the image tag, then opens a labeled `release-prep` PR.
- **`release-tag`** (triggered when that PR is merged) — tags the merge commit, builds
  binaries and multi-arch images, then publishes the GitHub Release.

The release type is detected from the version string passed to `release-prep`:

| Version input | Type | Base branch | Image-tag bump | Published as |
|---------------|------|-------------|----------------|--------------|
| `X.Y.Z-rc.N`  | Release candidate | `main` | no | pre-release |
| `X.Y.0`       | Minor | `release-vX.Y` | yes | latest |
| `X.Y.Z` (Z>0) | Patch | `release-vX.Y` | yes | latest |

> **Prerequisite:** because `pull_request` workflows run from the **base branch's** copy of
> the workflow file, `release-tag.yml` (plus the reusable `release.yml`/`docker.yml` and the
> portable `bump-tempo-image-tag` Makefile target) must exist on every branch that
> `release-prep` PRs target: `main` (for RCs) and each active `release-vX.Y` branch (for
> minor/patch). Backport these to active release branches before using the flow there.

## Steps

1. **Run `release-prep`.** Actions → `release-prep` → Run workflow, enter the version (no
   leading `v`, e.g. `3.1.0-rc.0`, `3.1.0`, or `3.0.3`). It validates the version, collates
   pending `.chloggen/` entries into `CHANGELOG.md`, sets the `VERSION` file, bumps the
   image tag + regenerates jsonnet for non-RC releases, and opens a `release-prep` PR
   against the correct base branch. The `validate_only` option dry-runs the checks without
   opening a PR.
2. **Review and merge the prep PR.** Merging is the human gate. **Merging auto-tags and
   publishes** — only merge when you intend to release.
3. **`release-tag` takes over** (triggered by the merge): it verifies the PR is a genuine
   release-prep PR (see Security below), tags the merge commit, builds binaries and
   multi-arch images, then un-drafts and publishes the GitHub Release — `--latest` for
   minor/patch, `--prerelease` for RCs — using the `CHANGELOG.md` section.
4. **Verify** the `release-tag` run succeeded and the images appear in GAR, then on Docker
   Hub after the mirror catches up.

## Security

`release-tag` triggers on any merged PR, so before tagging it verifies the PR is a genuine,
bot-generated release-prep PR and aborts otherwise:

- the PR was authored by the `tempo-ci-app` bot (a human cannot open a PR as the bot, so a
  forged PR is rejected even if it is correctly named, labeled, and merged);
- the `release-prep` label is present;
- the head branch matches `release-prep/vX.Y.Z[-rc.N]`;
- the base branch matches the release type (`main` for RCs, `release-vX.Y` otherwise).

The workflow that runs is always the base branch's copy of `release-tag.yml`, so the guard
cannot be weakened from within a PR.

## Steps that remain manual

- **Minor release changelog consolidation.** A minor (`X.Y.0`) is preceded by RCs whose
  `chlog-update` already consumed the `.chloggen/` entries, so there are usually none left
  to collate. Before running `release-prep` for the minor, rename the `# vX.Y.0-rc.N`
  heading(s) in `CHANGELOG.md` on the release branch to `# vX.Y.0`. `release-prep` verifies
  that section exists and fails with guidance if it doesn't.
- **Bump `VERSION` on `main` after cutting a release branch.** Submit a PR setting it to the
  next minor's dev version (e.g. `3.2.0-dev`); until it lands, main builds report the RC
  version left by the last RC prep.
- Release notes documentation, helm-chart PRs, and the changelog PR back to `main` (see the
  manual sections above).
