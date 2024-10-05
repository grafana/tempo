#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

is_valid_semver() {
      local version=$1
      # regex taken from https://semver.org/
      if [[ $version =~ ^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$ ]] ;then
            return 1
      else
            return 0
      fi
}


VERSION=$(git describe --tags `git rev-list --tags --max-count=1`)
if is_valid_semver "$VERSION"; then
      echo "$VERSION"
      exit 0
fi


source "${REPO_ROOT}/tools/image-tag"