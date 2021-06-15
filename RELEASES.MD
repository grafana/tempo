# Releases

This document details release procedures for Tempo.  Currently it's pretty dang easy.

- Push a semver tag to main.  Something like:
  - `git tag -a v0.3.0`
  - `git push origin v0.3.0`
- This will initiate the build process in Github Actions and Drone.  The tagged docker image should
  be available here shortly: https://hub.docker.com/r/grafana/tempo/tags?page=1&ordering=last_updated
- A Github Release Draft should also be available here:  https://github.com/grafana/tempo/releases
  - Copy over the CHANGELOG entries for the release
  - Call out contributors for their work
  - Cull unnecessary changes that don't impact the Tempo binary or deployment
  - Call out breaking changes!
- Submit a PR cleaning up the changelog and moving everything under "main/unreleased" to be under
  the newly minted version.
- Update docs
  - From the root of the website repo run `./scripts/docs-release.sh tempo v1.0.0 latest` 
    replacing v1.0.0 with the version and submit a PR.
- Update helm
  - Submit a PR to `github.com/grafana/helm-charts` to update to the newly cut version.

# Patch Releases

Patches release should be cut for serious bug fixes or security issues.

- Check to see if a release branch already exists. The release branch
  will match the same major and minor versions. If this branch doesn't exist then you will need to create it. See nested instructions.
  `git checkout release-v1.0.0`

  - Checkout to the release tag we are branching off. e.g.:  
    `git checkout v1.0.0`
  - Create a new branch. It should be named `release-<base version>`.  
    `git checkout -b release-v1.0.0`

- Cherry-pick the [commit hash](https://github.com/grafana/tempo/commits/main) that you want to apply to this release.  
   `git cherry-pick <commit hash>`
- Push the commit  
  `git push`
- Create and push a tag to initiate the release.  
  `git tag -a v1.0.1`  
  `git push origin v1.0.1`  
  This triggers the build.
- Follow CHANGELOG procedures above.
- Submit a PR to the Helm repo as described above.
