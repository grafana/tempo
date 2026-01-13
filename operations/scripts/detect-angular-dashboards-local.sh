#!/usr/bin/env bash
set -euo pipefail

show_help() {
  cat <<'EOF'
Local best-effort detection of legacy Angular panel types in dashboard JSON.

Usage:
  detect-angular-dashboards-local.sh [--include-table] [path ...]

If no paths are provided, the script checks:
  tempo-mixin-compiled/dashboards
  tempo-mixin-compiled-v3/dashboards

This script only scans JSON for known legacy panel types and cannot detect
custom/private Angular plugins without a running Grafana instance.
EOF
}

include_table=false
paths=()

for arg in "$@"; do
  case "$arg" in
    --include-table)
      include_table=true
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      paths+=("$arg")
      ;;
  esac
done

if [[ ${#paths[@]} -eq 0 ]]; then
  for candidate in tempo-mixin-compiled/dashboards tempo-mixin-compiled-v3/dashboards; do
    if [[ -d "$candidate" ]]; then
      paths+=("$candidate")
    fi
  done
fi

if [[ ${#paths[@]} -eq 0 ]]; then
  echo "No dashboard paths found. Provide paths or generate compiled dashboards." >&2
  exit 1
fi

legacy_types=("graph" "table-old" "singlestat" "grafana-singlestat-panel" "grafana-worldmap-panel" "grafana-piechart-panel")
if [[ "$include_table" == "true" ]]; then
  legacy_types+=("table")
fi

legacy_regex=$(printf "%s|" "${legacy_types[@]}")
legacy_regex=${legacy_regex%|}

matches=$(rg --pcre2 -n --glob '*.json' "\"type\"\\s*:\\s*\"(${legacy_regex})\"" "${paths[@]}" || true)
if [[ -z "$matches" ]]; then
  echo "No legacy panel types found in: ${paths[*]}"
  exit 0
fi

echo "Legacy panel type occurrences:"
echo "$matches"
echo
echo "Summary by panel type:"
echo "$matches" | sed -E 's/.*"type"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/' | sort | uniq -c | sort -nr

echo
echo "Panel titles by file:"
types_json=$(printf '%s\n' "${legacy_types[@]}" | jq -R . | jq -s .)
for path in "${paths[@]}"; do
  while IFS= read -r -d '' file; do
    titles=$(
      jq -r --argjson types "$types_json" '
        .. | select(type == "object")
        | .type? as $t
        | .title? as $title
        | select($t != null and $title != null)
        | select($types | index($t))
        | "\($t)\t\($title)"
      ' "$file"
    )
    if [[ -n "$titles" ]]; then
      echo "$file"
      echo "$titles" | sed 's/^/  - /'
    fi
  done < <(find "$path" -type f -name '*.json' -print0)
done
