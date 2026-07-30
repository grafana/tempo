#!/usr/bin/env bash
#
# Tests for tools/check-otel-proto-pin.sh. Run by .github/workflows/tools-tests.yml.

set -o nounset

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/tools" "$TMP/pkg/tempopb/.wiresmith-proto"
cp "$SCRIPT_DIR/check-otel-proto-pin.sh" "$TMP/tools/"

git -C "$TMP" init -q
git -C "$TMP" config user.email test@example.com
git -C "$TMP" config user.name test

PASS=0
FAIL=0

SHA_A="7c63f7b8b69e83bdda071a70898cd8a9f4ec77a2"
SHA_B="1111111111111111111111111111111111111111"

# Records a submodule gitlink at <sha> (committed - check-otel-proto-pin.sh
# reads it via `git ls-tree HEAD`, so it must be in a real commit, not just the
# working tree) and writes <pin file content> to OTEL_PROTO_PIN, uncommitted
# (the script reads that straight off disk).
set_state() { # <gitlink sha> <pin file content>
	git -C "$TMP" update-index --add --cacheinfo "160000,$1,opentelemetry-proto"
	git -C "$TMP" commit -q -m "set gitlink to $1" --allow-empty
	printf '%s\n' "$2" >"$TMP/pkg/tempopb/.wiresmith-proto/OTEL_PROTO_PIN"
}

expect_pass() { # <label>
	if out=$("$TMP/tools/check-otel-proto-pin.sh" 2>&1); then
		echo "PASS  $1"
		PASS=$((PASS + 1))
	else
		echo "FAIL  $1: expected success, got failure:"
		echo "    ${out//$'\n'/$'\n    '}"
		FAIL=$((FAIL + 1))
	fi
}

expect_fail() { # <label>
	if out=$("$TMP/tools/check-otel-proto-pin.sh" 2>&1); then
		echo "FAIL  $1: expected failure, script passed"
		FAIL=$((FAIL + 1))
	else
		echo "PASS  $1 (correctly failed)"
		PASS=$((PASS + 1))
	fi
}

set_state "$SHA_A" "$SHA_A"
expect_pass "matching pin"

set_state "$SHA_A" "$SHA_B"
expect_fail "mismatched pin"

set_state "$SHA_B" "$SHA_A"
expect_fail "gitlink moved without a matching pin bump (the original incident)"

set_state "$SHA_A" "# comment only, no sha"
expect_fail "pin file with no SHA"

set_state "$SHA_A" "  $SHA_A  "
expect_pass "pin file with surrounding whitespace"

printf '# %s\n%s\n' "leading comment" "$SHA_A" >"$TMP/pkg/tempopb/.wiresmith-proto/OTEL_PROTO_PIN"
expect_pass "pin file with a leading comment line"

rm "$TMP/pkg/tempopb/.wiresmith-proto/OTEL_PROTO_PIN"
expect_fail "missing pin file"

echo
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
