# Releases

This document details release procedures for Tempo.  Currently it's pretty dang easy.

- Push a semver tag to master.  Something like:
  - `git tag -a v0.3.0`
  - `git push origin --tags`
- This will initiate the build process in Github Actions and Drone.  The tagged docker iamge should
  be available here shortly: https://hub.docker.com/r/grafana/tempo/tags?page=1&ordering=last_updated
- A Github Release Draft should also be available here:  https://github.com/grafana/tempo/releases
  - Link the Changelog
  - Call out contributors for their work
  - Cull unnecessary changes that don't impact the Tempo binary or deployment
  - Call out breaking changes!
- Submit a PR cleaning up the changelog and moving everything under "master/unreleased" to be under
  the newly minted version.