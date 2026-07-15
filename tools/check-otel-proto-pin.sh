#!/usr/bin/env bash
#
# Verifies the opentelemetry-proto submodule pin (the commit tempo's tree
# points its gitlink at) matches pkg/tempopb/.wiresmith-proto/OTEL_PROTO_PIN,
# the commit the checked-in annotated OTLP proto snapshots
# (pkg/tempopb/.wiresmith-proto/{common,resource,trace}/v1/*.proto) were last
# manually re-derived from. Reads the gitlink straight out of the git tree
# (`git ls-tree`, not `git submodule`), so this needs no network access and no
# submodule checkout - safe to run as an early, cheap CI step.
#
# Without this gate the submodule pin can move (e.g. through a routine
# upstream rebase that carries the gitlink bump forward) while the checked-in
# snapshots silently stay stale: this already happened once, when
# opentelemetry-proto gained a new EntityRef message and tempo's generated
# types went a month without it while CI stayed green (gen-proto regenerates
# byte-identical output from the stale snapshot regardless of whether that
# snapshot still matches the submodule pin).

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SUBMODULE_PATH="opentelemetry-proto"
PIN_FILE="${REPO_ROOT}/pkg/tempopb/.wiresmith-proto/OTEL_PROTO_PIN"

cd "$REPO_ROOT"

if [[ ! -f "$PIN_FILE" ]]; then
	echo "error: recorded pin file not found: ${PIN_FILE}" >&2
	exit 1
fi

actual=$(git ls-tree HEAD -- "$SUBMODULE_PATH" | awk '{print $3}')
if [[ -z "$actual" ]]; then
	echo "error: 'git ls-tree HEAD -- ${SUBMODULE_PATH}' found no gitlink; is the submodule still declared at that path?" >&2
	exit 1
fi

recorded=$(grep -v '^#' "$PIN_FILE" | grep -v '^[[:space:]]*$' | head -n1 | tr -d '[:space:]')
if [[ -z "$recorded" ]]; then
	echo "error: ${PIN_FILE} has no recorded commit SHA (only comments/blank lines)" >&2
	exit 1
fi

if [[ "$actual" != "$recorded" ]]; then
	cat >&2 <<EOF
error: opentelemetry-proto submodule pin has moved without a matching proto-snapshot re-sync.

  submodule gitlink (HEAD):        ${actual}
  recorded snapshot pin (${PIN_FILE#"${REPO_ROOT}"/}): ${recorded}

The checked-in OTLP proto snapshots under pkg/tempopb/.wiresmith-proto/ are
supposed to be manually re-derived every time the submodule pin advances, and
the recorded pin bumped to match - but the two disagree right now.

To fix:
  1. Diff pkg/tempopb/.wiresmith-proto/**/*.proto against the current
     opentelemetry-proto submodule content (see the "Gen proto" section of
     'make gen-proto' in the Makefile for the copy/sed steps) and port any
     real message/field changes.
  2. Run 'make gen-proto' to regenerate the Go code.
  3. Update pkg/tempopb/.wiresmith-proto/OTEL_PROTO_PIN to ${actual}.
EOF
	exit 1
fi

echo "OK: opentelemetry-proto submodule pin (${actual}) matches the recorded snapshot pin."
