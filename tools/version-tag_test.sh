#!/usr/bin/env bash
#
# Tests for tools/version-tag.sh. Run by .github/workflows/tools-tests.yml.

set -o nounset

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
mkdir "$TMP/tools"
cp "$SCRIPT_DIR/version-tag.sh" "$TMP/tools/"

PASS=0
FAIL=0

expect_version() { # <VERSION file content> <expected output>
    printf '%s\n' "$1" > "$TMP/VERSION"
    out=$("$TMP/tools/version-tag.sh" 2>/dev/null)
    if [ "$out" = "$2" ]; then
        echo "PASS  '$1' -> '$2'"
        PASS=$((PASS + 1))
    else
        echo "FAIL  '$1': expected '$2', got '$out'"
        FAIL=$((FAIL + 1))
    fi
}

expect_error() { # <VERSION file content>
    printf '%s\n' "$1" > "$TMP/VERSION"
    if "$TMP/tools/version-tag.sh" >/dev/null 2>&1; then
        echo "FAIL  '$1' should be rejected, was accepted"
        FAIL=$((FAIL + 1))
    else
        echo "PASS  rejects '$1'"
        PASS=$((PASS + 1))
    fi
}

expect_version "2.10.6" "v2.10.6"
expect_version "3.1.0-dev" "v3.1.0-dev"
expect_version "3.0.0-rc.1" "v3.0.0-rc.1"
expect_version "1.2.3-rc.1+sha.974d430" "v1.2.3-rc.1+sha.974d430"

expect_error "v2.10.6"      # no leading v in the file
expect_error "01.2.3"       # leading zero
expect_error "1.2"          # missing patch
expect_error "1.2.3.4"      # extra component
expect_error "1.2.3-"       # empty prerelease
expect_error "main-927ed1b" # image-tag shape, not a version
expect_error ""

# missing VERSION file
rm "$TMP/VERSION"
if "$TMP/tools/version-tag.sh" >/dev/null 2>&1; then
    echo "FAIL  missing VERSION file should be an error"
    FAIL=$((FAIL + 1))
else
    echo "PASS  errors on missing VERSION file"
    PASS=$((PASS + 1))
fi

echo
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
