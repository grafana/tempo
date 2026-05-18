---
title: Reduce cardinality with span name sanitization
menuTitle: Reduce cardinality
description: Use DRAIN-based span name sanitization to reduce high-cardinality span name labels in generated metrics.
weight: 450
---

# Reduce cardinality with span name sanitization

{{< docs/experimental product="span name sanitization" >}}

The `span_name` label often contributes the most to high cardinality in generated metrics.
Applications that embed dynamic values in span names, for example, REST paths with user IDs (`GET /users/123`) or auto-generated operation names (`query-abc-def-ghi`), create a unique series for every distinct value.
This drives up active series counts, increases storage and query costs, and can push you past cardinality limits.

You can enable `span_name_sanitization` to solve this without changing instrumentation across your services.
The option uses the DRAIN algorithm to learn patterns from incoming span names and replace variable segments with a `<_>` placeholder, reducing many unique span names down to a single representative series.

Use span name sanitization when:

- Your `span_name` label has high cardinality due to embedded IDs, timestamps, or request parameters.
- You don't control the instrumentation that produces these span names, or changing it across all services isn't practical.
- You want to reclaim active series budget without losing meaningful aggregation in your span metrics.

## Before you begin

- The metrics-generator must be enabled and processing spans.
- You need access to the Tempo configuration or the [user-configurable overrides API](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/user-configurable-overrides/).

## How the DRAIN span name sanitization works

DRAIN is a streaming algorithm that learns patterns from span names as they arrive.
It breaks each span name into tokens (segments separated by delimiters like `/`, `-`, or spaces) and organizes them into a tree structure.
When the algorithm sees enough variation at a specific position, for example, numeric IDs, UUIDs, or other data-like values, it replaces that position with a `<_>` wildcard.

For example, given these span names:

- `GET /users/123`
- `GET /users/456`
- `GET /users/789`

DRAIN detects that the third token varies while the rest stays constant, and produces the sanitized pattern `GET /users/<_>`.
All three original span names map to this single pattern in generated metrics.

DRAIN preserves meaningful trailing segments.
For example, given `GET /users/123/data` and `GET /users/123/settings`, DRAIN sanitizes the ID but keeps `/data` and `/settings` distinct, producing `GET /users/<_>/data` and `GET /users/<_>/settings` rather than collapsing both paths into a single pattern.

The algorithm adapts continuously.
As new span names arrive, DRAIN refines its patterns and merges additional variable segments.
It also prunes stale patterns that haven't been seen recently.

For more details on the algorithm, refer to [Drain: An Online Log Parsing Approach with Fixed Depth Tree](https://ieeexplore.ieee.org/document/8029742/) (He et al., IEEE ICWS 2017).

## Configure the span name sanitizer

Set the `span_name_sanitization` option in the `overrides` block of your Tempo configuration:

```yaml
overrides:
  defaults:
    metrics_generator:
      span_name_sanitization: "enabled"
```

The `span_name_sanitization` option accepts these values:

| Value | Behavior |
|-------|----------|
| `""` (empty string) | Disabled. No sanitization is applied. This is the default. |
| `dry_run` | Trains the DRAIN model and produces the `tempo_metrics_generator_registry_post_sanitization_demand_estimate` metric, but doesn't modify span names in generated metrics. Use this mode to evaluate the cardinality impact before enabling. |
| `enabled` | Applies DRAIN sanitization to span names in generated metrics. |

You can also set this option per-tenant using the [user-configurable overrides API](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/user-configurable-overrides/).

## Evaluate the impact with dry-run mode

Before enabling `span_name_sanitization`, use `dry_run` mode to understand the impact without changing your metrics:

```yaml
overrides:
  defaults:
    metrics_generator:
      span_name_sanitization: "dry_run"
```

After the model has trained for a few minutes, compare the estimated demand against your current active series:

- `tempo_metrics_generator_registry_post_sanitization_demand_estimate` estimates cardinality if sanitization were fully applied
- `tempo_metrics_generator_registry_active_series` shows the current active series count

If the post-sanitization estimate is significantly lower, switching to `enabled` reduces your active series count.

## Monitor the span name sanitizer

After you enable `span_name_sanitization`, use these metrics to observe its effect:

- `tempo_metrics_generator_registry_spans_sanitized_total` counts the spans whose `span_name` label was replaced by a sanitized pattern
- `tempo_metrics_generator_registry_post_sanitization_demand_estimate` shows the ongoing cardinality demand after sanitization is applied

## Next steps

- [Cardinality](../cardinality/) for background on how cardinality impacts the metrics-generator
- [Estimate cardinality from traces](../estimate-cardinality/) to calculate expected active series for service graphs
- [Troubleshoot metrics-generator](https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/metrics-generator/) for diagnosing metrics quality issues
