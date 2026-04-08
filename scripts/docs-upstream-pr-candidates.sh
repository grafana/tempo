#!/usr/bin/env bash
# List merged PRs on grafana/tempo and open a triage issue on this repo (tempo-doc-work).
# Used by .github/workflows/docs-upstream-pr-candidates.yml — can also be run locally with
# GITHUB_REPOSITORY=owner/tempo-doc-work and gh auth login.

set -euo pipefail

UPSTREAM="${UPSTREAM_REPO:-grafana/tempo}"
DAYS="${SINCE_DAYS:-7}"
LABEL="${ISSUE_LABEL:-docs-triage}"
RUN_DATE=$(date -u +%Y-%m-%d)

if [[ "$(uname -s)" == "Darwin" ]]; then
	SINCE=$(date -u -v-${DAYS}d +%Y-%m-%d)
else
	SINCE=$(date -u -d "${DAYS} days ago" +%Y-%m-%d)
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

{
	echo "Merged pull requests on \`${UPSTREAM}\` since **${SINCE}** (generated **${RUN_DATE}** UTC)."
	echo ""
	echo "Use PR numbers with \`/docs-pr-check\` in your **tempo-doc-work** clone. Not every PR needs docs—skip test-only, vendor, or internal refactors."
	echo ""
	echo "| PR | Title |"
	echo "|----|-------|"
} >"$TMP"

JSON="$(gh pr list --repo "$UPSTREAM" --state merged \
	--search "merged:>=${SINCE}" \
	--json number,title,url \
	--limit 200 2>/dev/null || echo '[]')"

COUNT="$(echo "$JSON" | jq 'length')"

if [[ "$COUNT" -eq 0 ]]; then
	{
		echo "No merged PRs in \`${UPSTREAM}\` in the last **${DAYS}** days (since ${SINCE})."
		echo ""
		echo "Increase the window by setting \`SINCE_DAYS\` in the workflow or run \`workflow_dispatch\` after more merges."
	} >"$TMP"
else
	echo "$JSON" | jq -r '.[] | "| [#\(.number)](\(.url)) | \(.title | gsub("\\|"; "/") | gsub("\\n"; " ")) |"' >>"$TMP"
fi

TITLE="Upstream PRs: docs triage — ${RUN_DATE}"

if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
	echo "--- Preview (no issue created: GITHUB_REPOSITORY unset) ---"
	cat "$TMP"
	exit 0
fi

gh label create "$LABEL" --repo "$GITHUB_REPOSITORY" --color "FBCA04" --description "Upstream Tempo PRs for possible docs work" 2>/dev/null || true

if gh issue create --repo "$GITHUB_REPOSITORY" --title "$TITLE" --body-file "$TMP" --label "$LABEL"; then
	echo "Created issue: ${TITLE}"
else
	echo "gh issue create failed (permissions or API)." >&2
	exit 1
fi
