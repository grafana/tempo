#!/usr/bin/env bash
#
# Verifies pkg/tempopb/.wiresmith-proto/{common,resource,trace}/v1/*.proto
# still match the opentelemetry-proto submodule content structurally (message/
# field/oneof shape), modulo two known, deliberate divergence classes:
# comment-only rewording (historically left unported since it doesn't change
# generated code shape) and the wiresmith-specific additions (the
# wiresmith/options.proto import, the no_presence_all file option, and
# per-field "[(wiresmith.options.*) = ...]" annotations).
#
# This is a stronger, network-dependent complement to check-otel-proto-pin.sh:
# that script only catches a *moved* submodule pin with no matching snapshot
# re-sync; this one also catches a re-sync that bumped the recorded pin but
# ported the proto content incorrectly or incompletely. Requires the
# opentelemetry-proto submodule checkout (fetches it if not already present).

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$REPO_ROOT"

# Mirrors the copy/sed steps in the Makefile's gen-proto target - keep the two
# in sync if that target's patching rules ever change.
git submodule update --init

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

cp -R opentelemetry-proto/opentelemetry/proto/* "$TMP_DIR/"

UNAME=$(uname -s)
if [[ "$UNAME" == "Darwin" ]]; then
	SED_INPLACE=(-i '')
else
	SED_INPLACE=(-i)
fi

find "$TMP_DIR" -name '*.proto' -print0 | xargs -0 -L 1 sed "${SED_INPLACE[@]}" 's+ opentelemetry.proto+ tempopb+g'
find "$TMP_DIR" -name '*.proto' -print0 | xargs -0 -L 1 sed "${SED_INPLACE[@]}" 's+go.opentelemetry.io/proto/otlp+github.com/grafana/tempo/pkg/tempopb+g'
find "$TMP_DIR" -name '*.proto' -print0 | xargs -0 -L 1 sed "${SED_INPLACE[@]}" 's+import "opentelemetry/proto/+import "+g'

# Strips everything that's allowed to differ between the raw submodule source
# and our checked-in annotated snapshot, leaving only message/field/oneof
# structure to compare: comments, blank lines, file-level "option ...;"
# statements (namespace/package cosmetics), "import ...;" lines, and inline
# "[(wiresmith.options.*) = ...]" field annotations.
normalize() {
	sed -E \
		-e '/^[[:space:]]*\/\//d' \
		-e '/^[[:space:]]*$/d' \
		-e '/^[[:space:]]*option[[:space:]]/d' \
		-e '/^[[:space:]]*import[[:space:]]/d' \
		-e 's/[[:space:]]*\[[^]]*\][[:space:]]*;/;/' \
		-e 's/^[[:space:]]+//' \
		-e 's/[[:space:]]+$//' \
		"$1" | awk '{print}'
}

SNAPSHOT_FILES=(
	common/v1/common.proto
	resource/v1/resource.proto
	trace/v1/trace.proto
)

status=0
for f in "${SNAPSHOT_FILES[@]}"; do
	if [[ ! -f "$TMP_DIR/$f" ]]; then
		echo "error: opentelemetry-proto submodule no longer has opentelemetry/proto/${f}; pkg/tempopb/.wiresmith-proto/${f} needs manual attention (file moved/removed upstream)." >&2
		status=1
		continue
	fi
	if ! diff -u <(normalize "$TMP_DIR/$f") <(normalize "pkg/tempopb/.wiresmith-proto/$f") >"$TMP_DIR/$(basename "$f").diff"; then
		echo "error: pkg/tempopb/.wiresmith-proto/${f} has drifted from the opentelemetry-proto submodule content (structural diff below; comments/options/wiresmith annotations already stripped from both sides):" >&2
		cat "$TMP_DIR/$(basename "$f").diff" >&2
		status=1
	fi
done

if [[ "$status" -ne 0 ]]; then
	cat >&2 <<EOF

Re-derive the annotated snapshot(s) flagged above from the current
opentelemetry-proto submodule content (see the "Gen proto" section of
'make gen-proto' in the Makefile for the copy/sed steps this script mirrors),
port the real changes, run 'make gen-proto', then update
pkg/tempopb/.wiresmith-proto/OTEL_PROTO_PIN.
EOF
	exit 1
fi

echo "OK: pkg/tempopb/.wiresmith-proto/*.proto match the opentelemetry-proto submodule content (modulo wiresmith annotations)."
