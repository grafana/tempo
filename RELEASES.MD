# Releases

This document details release procedures for Tempo.  Currently it's pretty dang easy.

- Push a semver tag to main.  Something like:
  - `git tag -a v0.3.0`
  - `git push origin v0.3.0`
- This will initiate the build process in Github Actions and Drone.  The tagged docker image should
  be available here shortly: https://hub.docker.com/r/grafana/tempo/tags?page=1&ordering=last_updated
- A Github Release Draft should also be available here:  https://github.com/grafana/tempo/releases
  - Copy over the CHANGELOG entries for the release
  - Add the github handles of contributors at the end of each entry
  - Call out contributors for their work
  - Cull unnecessary changes that don't impact the Tempo binary or deployment
  - Call out breaking changes!
- Submit a PR cleaning up the changelog and moving everything under "main/unreleased" to be under
  the newly minted version.
- If any changes were made to the manifest, update [the docs as well](https://grafana.com/docs/tempo/latest/configuration/manifest/).