#!/usr/bin/env bash
#
# Prints the build version (e.g. v3.1.0, v3.1.0-rc.0, v3.1.0-dev) read from the VERSION file at
# the repo root, which is the single source of truth for the version baked into
# binaries. Release tags must match it; docker.yml and release.yml enforce this
# in CI. Reading the version from checkout content instead of the git tag list
# keeps it correct in shallow clones and under concurrent releases.

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

VERSION="v$(<"${REPO_ROOT}/VERSION")"

# semver.org regex as POSIX ERE (the VERSION file holds the version without the v prefix)
SEMVER='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$'
if [[ ! $VERSION =~ $SEMVER ]]; then
      echo "invalid semver '${VERSION#v}' in ${REPO_ROOT}/VERSION" >&2
      exit 1
fi

echo "$VERSION"
