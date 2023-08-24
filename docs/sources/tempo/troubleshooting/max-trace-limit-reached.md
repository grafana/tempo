---
title: Distributor refusing spans
description: Troubleshoot distributor refusing spans
weight: 471
aliases:
- ../operations/troubleshooting/max-trace-limit-reached/
---

# Distributor refusing spans

The two most likely causes of refused spans are unhealthy ingesters or trace limits being exceeded.

## Unhealthy ingesters

Unhealthy ingesters can be caused by failing OOMs or scale down events.
If you have unhealthy ingesters, your log line will look something like this:

```
msg="pusher failed to consume trace data" err="at least 2 live replicas required, could only find 1"
```

In this case, you may need to visit the ingester [ring page]({{< relref "../operations/consistent_hash_ring" >}}) at `/ingester/ring` on the Distributors
and "Forget" the unhealthy ingesters. This will work in the short term, but the long term fix is to stabilize your ingesters.

## Trace limits reached

In high volume tracing environments, the default trace limits are sometimes not sufficient.
These limits exist to protect Tempo and prevent it from OOMing, crashing or otherwise allow tenants to not DOS each other.
If you are refusing spans due to limits, you will see logs like this at the distributor:

```
msg="pusher failed to consume trace data" err="rpc error: code = FailedPrecondition desc = TRACE_TOO_LARGE: max size of trace (52428800) exceeded while adding 15632 bytes to trace a0fbd6f9ac5e2077d90a19551dd67b6f for tenant single-tenant"
msg="pusher failed to consume trace data" err="rpc error: code = FailedPrecondition desc = LIVE_TRACES_EXCEEDED: max live traces per tenant exceeded: per-user traces limit (local: 60000 global: 0 actual local: 60000) exceeded"
msg="pusher failed to consume trace data" err="rpc error: code = ResourceExhausted desc = RATE_LIMITED: ingestion rate limit (15000000 bytes) exceeded while adding 10 bytes"
```

You will also see the following metric incremented. The `reason` label on this metric will contain information about the refused reason.

```
tempo_discarded_spans_total
```

In this case, use available configuration options to [increase limits]({{< relref "../configuration#ingestion-limits" >}}).
