---
title: Distributor refusing spans
description: Troubleshoot distributor refusing spans
weight: 471
aliases:
  - ../../operations/troubleshooting/max-trace-limit-reached/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/troubleshooting/max-trace-limit-reached/
  - ../max-trace-limit-reached/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/max-trace-limit-reached/
---

# Distributor refusing spans

The most likely cause of refused spans is rate limits being exceeded.

To size your ingestion limits proactively and identify what's driving trace volume, refer to [Manage trace ingestion](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-trace-ingestion/).

To log spans that are discarded, add the `--distributor.log-discarded-spans.enabled` flag to the distributor or
adjust the [distributor configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#distributor):

```yaml
distributor:
  log_discarded_spans:
    enabled: true
    include_all_attributes: false # set to true for more verbose logs
```

Adding the flag logs all discarded spans, as shown below:

```
level=info ts=2024-08-19T16:06:25.880684385Z caller=distributor.go:767 msg=discarded spanid=c2ebe710d2e2ce7a traceid=bd63605778e3dbe935b05e6afd291006
level=info ts=2024-08-19T16:06:25.881169385Z caller=distributor.go:767 msg=discarded spanid=5352b0cb176679c8 traceid=ba41cae5089c9284e18bca08fbf10ca2
```

## Rate limits exceeded

The distributor checks ingestion rate limits before writing to Kafka. If spans are refused due to rate limits, you'll see logs like this at the distributor:

```
msg="pusher failed to consume trace data" err="rpc error: code = ResourceExhausted desc = RATE_LIMITED: ingestion rate limit (30000000 bytes) exceeded while adding 10 bytes"
```

You'll also see the following metric incremented. The `reason` label on this metric will contain information about the refused reason.

```
tempo_discarded_spans_total
```

In this case, use available configuration options to [increase limits](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#ingestion-limits).

## Trace limits

Limits such as `max_bytes_per_trace` and `max_live_traces_bytes` are enforced asynchronously by the live-store and
block-builder. These limits won't cause the distributor to refuse spans at ingestion time. Traces that exceed them are discarded downstream.

## Client resets connection

When the client resets the connection before the distributor can consume the trace data, you see logs like this:

```
msg="pusher failed to consume trace data" err="context canceled"
```

This issue needs to be fixed on the client side. To inspect which clients are causing the issue, logging discarded spans
with `include_all_attributes: true` may help.

Note that there may be other reasons for a closed context as well. Identifying the reason for a closed context is
not straightforward and may require additional debugging.
