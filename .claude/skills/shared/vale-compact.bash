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

if [[ "$#" -eq 0 ]]; then
  echo "usage: vale-compact.bash <file>..." >&2
  exit 2
fi

if ! command -v jq >/dev/null 2>&1; then
  exec vale "$@"
fi

json="$(vale --no-exit --output=JSON "$@")"

printf '%s\n' "$json" | jq -r '
  [
    to_entries[] as $f
    | $f.value[] as $a
    | {
        file: $f.key,
        line: $a.Line,
        col: $a.Span[0],
        severity: $a.Severity,
        check: $a.Check,
        match: $a.Match
      }
  ]
  | sort_by(.file, .line, .col)
  | .[]
  | "\(.file):\(.line):\(.col)\t\(.severity)\t\(.check)\t\(.match)"
'

printf '\n'
printf '%s\n' "$json" | jq -r '[.[][] | .Check] | group_by(.) | map("\(length)\t\(.[0])") | sort | .[]'
printf '%s\n' "$json" | jq -e '[.[][] | select(.Severity=="error")] | length == 0' >/dev/null
