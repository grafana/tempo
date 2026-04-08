#!/usr/bin/env bash
# Run mechanical docs checks (Markdown links) for local scheduling (cron / launchd).
# Same commands can be copied into a GitHub Actions job for grafana/tempo later.
#
# Usage:
#   ./docs/docs-assistant/run-local-checks.sh
#   DOCS_DIR=docs/sources/tempo ./docs/docs-assistant/run-local-checks.sh   # narrower scope
#
# Environment:
#   DOCS_DIR  Path to Markdown root (default: repo's docs/sources)
#   PYTHON    Python interpreter (default: python3)
#   LINK_CHECK_STRICT  If set to "1", run links.py without --grafana-hugo (noisy on Tempo sources)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCS_DIR="${DOCS_DIR:-$REPO_ROOT/docs/sources}"
PYTHON="${PYTHON:-python3}"
LINKS="$SCRIPT_DIR/links.py"
GRAFANA_HUGO_ARGS=(--grafana-hugo)
if [[ "${LINK_CHECK_STRICT:-}" == "1" ]]; then
	GRAFANA_HUGO_ARGS=()
fi

if [[ ! -d "$DOCS_DIR" ]]; then
	echo "ERROR: DOCS_DIR not found: $DOCS_DIR" >&2
	exit 1
fi

if [[ ! -f "$LINKS" ]]; then
	echo "ERROR: links.py not found: $LINKS" >&2
	exit 1
fi

echo "=== $(date -u +"%Y-%m-%dT%H:%M:%SZ") tempo-doc-work local docs checks ==="
echo "REPO_ROOT=$REPO_ROOT"
echo "DOCS_DIR=$DOCS_DIR"
echo ""

exec "$PYTHON" "$LINKS" "${GRAFANA_HUGO_ARGS[@]}" "$DOCS_DIR"
