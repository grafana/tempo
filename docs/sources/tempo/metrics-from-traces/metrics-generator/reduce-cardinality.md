---
title: Reduce cardinality with span name clustering
menuTitle: Reduce cardinality
description: Use DRAIN-based span name clustering to reduce high-cardinality span name labels in generated metrics.
weight: 450
---

# Reduce cardinality with span name clustering

The `span_name` label is often the largest contributor to high cardinality in generated metrics.
Applications that include dynamic values in span names create a unique series for every distinct value.
The `span_name_sanitization` option clusters similar span names together, replacing variable segments with a `<_>` placeholder to reduce active series.

## Before you begin

- The metrics-generator must be enabled and processing spans.
- You need access to the Tempo configuration or the [user-configurable overrides API](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/user-configurable-overrides/).

## How DRAIN clustering works

DRAIN is a streaming algorithm that learns patterns from span names as they arrive.
It breaks each span name into tokens (segments separated by delimiters like `/`, `-`, or spaces) and organizes them into a tree structure.
When the algorithm sees enough variation at a specific position, for example, numeric IDs, UUIDs, or other data-like values, it replaces that position with a `<_>` wildcard.

For example, given these span names:

- `GET /users/123`
- `GET /users/456`
- `GET /users/789`

DRAIN detects that the third token varies while the rest stays constant, and produces the pattern `GET /users/<_>`.
All three original span names map to this single pattern in generated metrics.

The algorithm adapts continuously.
As new span names arrive, DRAIN refines its patterns and merges additional variable segments.
It also prunes stale patterns that haven't been seen recently.

For more details on the algorithm, refer to [Drain: An Online Log Parsing Approach with Fixed Depth Tree](https://ieeexplore.ieee.org/document/8029742/) (He et al., IEEE ICWS 2017).

## Configure span name clustering

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
| `""` (empty string) | Disabled. No clustering is applied. This is the default. |
| `dry_run` | Trains the clustering model and produces the `tempo_metrics_generator_registry_post_sanitization_demand_estimate` metric, but doesn't modify span names in generated metrics. Use this mode to evaluate the cardinality impact before enabling. |
| `enabled` | Applies DRAIN clustering to span names in generated metrics. |

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

- `tempo_metrics_generator_registry_post_sanitization_demand_estimate` estimates cardinality if clustering were fully applied
- `tempo_metrics_generator_registry_active_series` shows the current active series count

If the post-clustering estimate is significantly lower, switching to `enabled` reduces your active series count.

## Monitor span name clustering

After you enable `span_name_sanitization`, use these metrics to observe its effect:

- `tempo_metrics_generator_registry_spans_sanitized_total` counts the spans whose `span_name` label was replaced by a clustered pattern
- `tempo_metrics_generator_registry_post_sanitization_demand_estimate` shows the ongoing cardinality demand after clustering is applied

## Next steps

- [Cardinality](../cardinality/) for background on how cardinality impacts the metrics-generator
- [Estimate cardinality from traces](../estimate-cardinality/) to calculate expected active series for service graphs
- [Troubleshoot metrics-generator](https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/metrics-generator/) for diagnosing metrics quality issues
