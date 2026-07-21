#!/usr/bin/env bash
# Full-stack smoke test for the bloom gateway's write path: distributor ->
# Kafka -> block-builder -> object store, backend-scheduler/backend-worker
# compaction+retention, and the bloom-gateway consuming the resulting
# publish/compaction/retention events. Does NOT build any image -- run
# `make docker-tempo` from the repo root first (see readme.md).
#
# Usage: ./smoke.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

COMPOSE=(docker compose)
IMAGE_TAG="${TEMPO_IMAGE_TAG:-latest}"

BB_METRICS="http://localhost:13210/metrics"
BW_METRICS="http://localhost:13230/metrics"
BG_METRICS="http://localhost:13240/metrics"

# Overall wall-clock budget for every polling assertion combined (not
# counting `docker compose up -d` itself). Each assertion also has its own
# tighter timeout below; whichever fires first wins.
OVERALL_BUDGET_SECONDS=480
start_ts=$(date +%s)
deadline_ts=$((start_ts + OVERALL_BUDGET_SECONDS))

FAILED=0

# ---------------------------------------------------------------------------
# cleanup
# ---------------------------------------------------------------------------
cleanup() {
	local exit_code=$?
	if [ "$exit_code" -ne 0 ] || [ "$FAILED" -ne 0 ]; then
		echo ""
		echo "=== compose ps (at failure) ==="
		"${COMPOSE[@]}" ps
	fi
	echo ""
	echo "=== tearing down: docker compose down -v ==="
	"${COMPOSE[@]}" down -v
}
trap cleanup EXIT

fail() {
	# fail SERVICE MESSAGE
	local service="$1" message="$2"
	FAILED=1
	echo ""
	echo "FAIL: $message"
	if [ -n "$service" ]; then
		echo "--- docker compose logs --tail 50 $service ---"
		"${COMPOSE[@]}" logs --tail 50 "$service"
	fi
	exit 1
}

# ---------------------------------------------------------------------------
# metrics helpers
# ---------------------------------------------------------------------------

# get_metric_sum URL METRIC_NAME [LABEL_SUBSTRING]
# Sums the value of every exposition line whose metric name matches exactly
# (summing across label combinations, e.g. a CounterVec), optionally
# filtered to lines containing LABEL_SUBSTRING (e.g. 'result="ok"'). Prints
# "0" if the metric is absent or the endpoint is unreachable -- never fails
# the caller.
get_metric_sum() {
	local url="$1" metric="$2" filter="${3:-}"
	local lines
	lines=$(curl -fsS --max-time 5 "$url" 2>/dev/null | grep -E "^${metric}(\{[^}]*\})? ")
	if [ -n "$filter" ] && [ -n "$lines" ]; then
		lines=$(printf '%s\n' "$lines" | grep -F "$filter" || true)
	fi
	if [ -z "$lines" ]; then
		echo "0"
		return
	fi
	printf '%s\n' "$lines" | awk '{sum += $NF} END{printf "%d\n", sum}'
}

# check_reconciliation_repairs_zero: assertion 5. Sampled on every polling
# iteration of every other assertion below (not just once at the end) --
# reconciliation repairs are the producer-correctness canary: a repair means
# a producer publish went missing and the safety net had to patch the
# gateway's view itself, which the smoke must treat as a hard failure the
# instant it happens, not just at the end.
check_reconciliation_repairs_zero() {
	local sum
	sum=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_reconciliation_repairs_total" "")
	if [ "$sum" != "0" ]; then
		fail "bloom-gateway-0" "[5/6] tempo_bloom_gateway_reconciliation_repairs_total became $sum (want 0 for the entire run) -- a producer publish went missing and reconciliation had to repair the gateway's view"
	fi
}

# wait_for DESC TIMEOUT_SECONDS SERVICE CHECK_FN
# Polls CHECK_FN every 5s until it succeeds, TIMEOUT_SECONDS elapses, or the
# overall smoke budget is exhausted -- whichever comes first.
wait_for() {
	local desc="$1" timeout="$2" service="$3" check="$4"
	local step_start
	step_start=$(date +%s)
	while true; do
		check_reconciliation_repairs_zero
		if "$check"; then
			echo "PASS: $desc (after $(($(date +%s) - step_start))s)"
			return 0
		fi
		local now
		now=$(date +%s)
		if [ $((now - step_start)) -ge "$timeout" ]; then
			fail "$service" "$desc: timed out after ${timeout}s"
		fi
		if [ "$now" -ge "$deadline_ts" ]; then
			fail "$service" "$desc: overall ~${OVERALL_BUDGET_SECONDS}s smoke budget exceeded"
		fi
		sleep 5
	done
}

# ---------------------------------------------------------------------------
# assertion checks
# ---------------------------------------------------------------------------

check_bb_publishes() {
	local v
	v=$(get_metric_sum "$BB_METRICS" "tempo_bloom_gateway_publishes_total" 'result="ok"')
	[ "$v" -ge 1 ]
}
check_gateway_live() {
	local live entries
	live=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_blocks_live" "")
	entries=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_entries_total" "")
	[ "$live" -ge 1 ] && [ "$entries" -gt 0 ]
}
check_bw_publishes() {
	local v
	v=$(get_metric_sum "$BW_METRICS" "tempo_bloom_gateway_publishes_total" 'result="ok"')
	[ "$v" -ge 1 ]
}
check_gateway_deletes() {
	local v
	v=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_deletes_total" "")
	[ "$v" -ge 1 ]
}
check_gateway_reachable_and_live() {
	curl -fsS --max-time 5 "$BG_METRICS" >/dev/null 2>&1 || return 1
	local live
	live=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_blocks_live" "")
	[ "$live" -ge 1 ]
}

# ---------------------------------------------------------------------------
# preflight
# ---------------------------------------------------------------------------
if ! docker info >/dev/null 2>&1; then
	echo "FAIL: docker daemon is not available" >&2
	exit 1
fi

if [ -z "$(docker images -q "grafana/tempo:${IMAGE_TAG}" 2>/dev/null)" ]; then
	echo "FAIL: grafana/tempo:${IMAGE_TAG} not found locally." >&2
	echo "Build it first from the repo root: make docker-tempo" >&2
	exit 1
fi

echo "=== docker compose up -d (image grafana/tempo:${IMAGE_TAG}) ==="
if ! "${COMPOSE[@]}" up -d; then
	fail "" "docker compose up -d failed"
fi
"${COMPOSE[@]}" ps

echo ""
echo "=== [1/6] block-builder: tempo_bloom_gateway_publishes_total{result=\"ok\"} >= 1 ==="
wait_for "[1/6] block-builder publish" 90 "block-builder-0" check_bb_publishes

echo ""
echo "=== [2/6] bloom-gateway: blocks_live >= 1 and entries_total > 0 (consumed + committed) ==="
wait_for "[2/6] gateway consumed+committed" 60 "bloom-gateway-0" check_gateway_live

echo ""
echo "=== [3/6] backend-worker: tempo_bloom_gateway_publishes_total{result=\"ok\"} >= 1 (compaction Add, needs >=2 input blocks) ==="
wait_for "[3/6] backend-worker compaction publish" 240 "backend-worker-0" check_bw_publishes

echo ""
echo "=== [4/6] bloom-gateway: deletes_total >= 1 (retention Delete round trip) ==="
wait_for "[4/6] gateway deletes applied" 240 "bloom-gateway-0" check_gateway_deletes

check_reconciliation_repairs_zero
echo "PASS: [5/6] reconciliation_repairs_total stayed 0 for the entire run so far"

echo ""
echo "=== [6/6] restart resilience: restart bloom-gateway-0, expect blocks_live to recover within 2m ==="
pre_restart_live=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_blocks_live" "")
pre_restart_snapshot_age=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_snapshot_age_seconds" "")
echo "blocks_live before restart: $pre_restart_live (snapshot_age_seconds: $pre_restart_snapshot_age)"

if ! "${COMPOSE[@]}" restart bloom-gateway-0; then
	fail "bloom-gateway-0" "[6/6] docker compose restart bloom-gateway-0 failed"
fi

wait_for "[6/6] gateway recovered after restart" 120 "bloom-gateway-0" check_gateway_reachable_and_live
check_reconciliation_repairs_zero

# ---------------------------------------------------------------------------
# summary
# ---------------------------------------------------------------------------
final_bb_publishes=$(get_metric_sum "$BB_METRICS" "tempo_bloom_gateway_publishes_total" 'result="ok"')
final_bw_publishes=$(get_metric_sum "$BW_METRICS" "tempo_bloom_gateway_publishes_total" 'result="ok"')
final_live=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_blocks_live" "")
final_entries=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_entries_total" "")
final_deletes=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_deletes_total" "")
final_repairs=$(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_reconciliation_repairs_total" "")
total_elapsed=$(($(date +%s) - start_ts))

echo ""
echo "=================================================================="
echo "PASS: bloom-gateway docker-compose smoke test (${total_elapsed}s total)"
echo "  [1/6] block-builder publishes_total{ok}   = $final_bb_publishes"
echo "  [2/6] gateway blocks_live / entries_total  = $final_live / $final_entries"
echo "  [3/6] backend-worker publishes_total{ok}   = $final_bw_publishes"
echo "  [4/6] gateway deletes_total                = $final_deletes"
echo "  [5/6] gateway reconciliation_repairs_total = $final_repairs (held at 0 for the entire run)"
echo "  [6/6] gateway blocks_live after restart    = $(get_metric_sum "$BG_METRICS" "tempo_bloom_gateway_blocks_live" "")"
echo "=================================================================="
