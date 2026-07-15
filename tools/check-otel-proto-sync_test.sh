#!/usr/bin/env bash
#
# Tests for tools/check-otel-proto-sync.sh. Run by
# .github/workflows/tools-tests.yml. Uses small fixture .proto content instead
# of the real opentelemetry-proto submodule, so this needs no network access
# and runs fast.

set -o nounset

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/tools"
cp "$SCRIPT_DIR/check-otel-proto-sync.sh" "$TMP/tools/"
git -C "$TMP" init -q

PASS=0
FAIL=0

SNAPSHOT_FILES=(
	common/v1/common.proto
	resource/v1/resource.proto
	trace/v1/trace.proto
)

# Writes a matching submodule/snapshot pair for <rel path>: same message/field
# shape, differing only in the ways the real gen-proto pipeline and
# check-otel-proto-sync.sh's normalizer are expected to tolerate (package
# rename via the sed rule under test, comment rewording, the wiresmith import/
# option/annotation additions).
write_matching_pair() { # <rel path>
	local rel="$1"
	local sub="$TMP/opentelemetry-proto/opentelemetry/proto/$rel"
	local snap="$TMP/pkg/tempopb/.wiresmith-proto/$rel"
	mkdir -p "$(dirname "$sub")" "$(dirname "$snap")"
	cat >"$sub" <<'EOF'
// Copyright Test Authors
syntax = "proto3";
package opentelemetry.proto.test.v1;
option go_package = "go.opentelemetry.io/proto/otlp/test/v1";

// Widget represents a thing.
message Widget {
  // The name of the widget.
  string name = 1;
  // A nested widget.
  Widget other = 2;
}
EOF
	cat >"$snap" <<'EOF'
// Copyright Test Authors
syntax = "proto3";
package tempopb.test.v1;
import "wiresmith/options.proto";
option (wiresmith.options.no_presence_all) = true;
option go_package = "github.com/grafana/tempo/pkg/tempopb/test/v1;v1";

// Widget is a thing of interest (reworded, allowed to lag upstream).
message Widget {
  // name of the widget.
  string name = 1;
  Widget other = 2 [(wiresmith.options.pointer) = true];
}
EOF
}

reset_matching_fixture() {
	rm -rf "$TMP/opentelemetry-proto" "$TMP/pkg"
	for rel in "${SNAPSHOT_FILES[@]}"; do
		write_matching_pair "$rel"
	done
}

expect_pass() { # <label>
	if out=$("$TMP/tools/check-otel-proto-sync.sh" 2>&1); then
		echo "PASS  $1"
		PASS=$((PASS + 1))
	else
		echo "FAIL  $1: expected success, got failure:"
		echo "    ${out//$'\n'/$'\n    '}"
		FAIL=$((FAIL + 1))
	fi
}

expect_fail() { # <label> <substring expected in output>
	if out=$("$TMP/tools/check-otel-proto-sync.sh" 2>&1); then
		echo "FAIL  $1: expected failure, script passed:"
		echo "    ${out//$'\n'/$'\n    '}"
		FAIL=$((FAIL + 1))
	elif [[ "$out" != *"$2"* ]]; then
		echo "FAIL  $1: failed as expected, but output didn't mention '$2':"
		echo "    ${out//$'\n'/$'\n    '}"
		FAIL=$((FAIL + 1))
	else
		echo "PASS  $1 (correctly failed, mentioning '$2')"
		PASS=$((PASS + 1))
	fi
}

reset_matching_fixture
expect_pass "all three snapshots structurally match (modulo comments/options/annotations)"

reset_matching_fixture
# Simulate the original incident: upstream adds a field, the snapshot doesn't
# get it.
cat >"$TMP/opentelemetry-proto/opentelemetry/proto/resource/v1/resource.proto" <<'EOF'
// Copyright Test Authors
syntax = "proto3";
package opentelemetry.proto.test.v1;
option go_package = "go.opentelemetry.io/proto/otlp/test/v1";

// Widget represents a thing.
message Widget {
  // The name of the widget.
  string name = 1;
  // A nested widget.
  Widget other = 2;
  // A brand new upstream field the snapshot hasn't been re-synced for.
  string new_upstream_field = 3;
}
EOF
expect_fail "unported new upstream field in resource.proto" "pkg/tempopb/.wiresmith-proto/resource/v1/resource.proto has drifted"

reset_matching_fixture
rm "$TMP/opentelemetry-proto/opentelemetry/proto/trace/v1/trace.proto"
expect_fail "submodule file removed/moved upstream" "no longer has opentelemetry/proto/trace/v1/trace.proto"

reset_matching_fixture
# A pure comment reword upstream (no field/message change) must NOT trip the
# gate - only structural drift should.
cat >"$TMP/opentelemetry-proto/opentelemetry/proto/common/v1/common.proto" <<'EOF'
// Copyright Test Authors
syntax = "proto3";
package opentelemetry.proto.test.v1;
option go_package = "go.opentelemetry.io/proto/otlp/test/v1";

// Widget represents an updated thing (wording only, no shape change).
message Widget {
  // The name of the widget, now explained slightly differently.
  string name = 1;
  // A nested widget, comment reworded upstream.
  Widget other = 2;
}
EOF
expect_pass "upstream comment-only reword is tolerated"

echo
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
