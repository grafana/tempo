#!/usr/bin/env bash
# Compact Vale output: same findings as `vale <file>...`, without the
# ~130-word boilerplate paragraph Vale repeats per finding regardless of
# --output template. Falls back to raw vale (or a warning) if jq/vale
# aren't installed.
set -euo pipefail

if ! command -v vale >/dev/null 2>&1; then
  echo "vale not installed (https://vale.sh/docs/install/); check style manually." >&2
  exit 0
fi
if [ "$#" -eq 0 ]; then
  echo "usage: vale-compact.sh <file>..." >&2
  exit 1
fi

json="$(vale --output=JSON "$@" || true)"

if ! command -v jq >/dev/null 2>&1; then
  echo "$json"
  exit 0
fi

echo "$json" | jq -r '
  to_entries[] as $f
  | $f.value[] as $a
  | [$f.key, ($a.Line|tostring), ($a.Span[0]|tostring), $a.Severity, $a.Check, $a.Match] | @tsv
' | sort -t $'\t' -k1,1 -k2,2n | awk -F'\t' '{printf "%s:%s:%s\t%s\t%s\t%s\n",$1,$2,$3,$4,$5,$6}'

echo
echo "$json" | jq -r '[.[][] | .Check] | group_by(.) | map("\(length)\t\(.[0])") | sort | .[]'

echo "$json" | jq -e '[.[][] | select(.Severity=="error")] | length == 0' >/dev/null
