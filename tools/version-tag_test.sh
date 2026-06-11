#!/usr/bin/env bash
#
# Tests for is_valid_semver in tools/version-tag.sh

set -o nounset

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1090
source <(sed -n '/^is_valid_semver/,/^}/p' "$SCRIPT_DIR/version-tag.sh")

PASS=0
FAIL=0

expect_valid() {
    if is_valid_semver "$1"; then
        echo "PASS  accepts '$1'"
        PASS=$((PASS + 1))
    else
        echo "FAIL  '$1' should be accepted, was rejected"
        FAIL=$((FAIL + 1))
    fi
}

expect_invalid() {
    if is_valid_semver "$1"; then
        echo "FAIL  '$1' should be rejected, was accepted"
        FAIL=$((FAIL + 1))
    else
        echo "PASS  rejects '$1'"
        PASS=$((PASS + 1))
    fi
}

# valid versions
expect_valid "v2.10.6" # multi-digit component
expect_valid "v2.9.3"
expect_valid "v0.1.0"
expect_valid "v3.0.0-rc.1" # prerelease
expect_valid "v1.0.0-rc.0"
expect_valid "v1.2.3-alpha.beta-x.7"
expect_valid "v1.2.3+build.42" # build metadata
expect_valid "v1.2.3-rc.1+sha.974d430"

# invalid versions
expect_invalid "v11234" # no dots
expect_invalid "abc"
expect_invalid "HEAD-abc-wip" # shape of the image-tag fallback output
expect_invalid "main-927ed1b"
expect_invalid "v01.2.3"  # leading zero
expect_invalid "v1.2"     # missing patch
expect_invalid "v1.2.3.4" # extra component
expect_invalid "2.10.6"   # missing v prefix
expect_invalid "v1.2.3-"  # empty prerelease
expect_invalid " v1.2.3"  # leading whitespace
expect_invalid ""

echo
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
