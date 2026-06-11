#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

is_valid_semver() {
      local version=$1
      # regex taken from https://semver.org/
      if echo "$version" | grep -qP '^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$' ;then
            return 0
      else
            return 1
      fi
}


# 1. Release build: HEAD itself is tagged — that tag is the version.
VERSION=$(git describe --exact-match --tags HEAD 2>/dev/null) ||
# 2. Build from an untagged commit: nearest release tag in HEAD's ancestry.
VERSION=$(git describe --tags --abbrev=0 2>/dev/null) ||
# 3. Shallow clone (no ancestry to walk): newest tag in the repo by commit date.
VERSION=$(git describe --tags "$(git rev-list --tags --max-count=1)" 2>/dev/null) ||
VERSION=""

if [ -n "$VERSION" ] && is_valid_semver "$VERSION"; then
      echo "$VERSION"
      exit 0
fi


source "${REPO_ROOT}/tools/image-tag"