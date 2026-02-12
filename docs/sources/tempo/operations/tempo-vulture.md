---
title: Tempo Vulture
description: Guide to using Tempo Vulture for data integrity testing
keywords:
  ["tempo", "vulture", "tempo-vulture", "testing", "validation", "monitoring"]
weight: 850
---

# Tempo Vulture

Tempo Vulture is a testing tool that validates the end-to-end functionality of Tempo by continuously writing traces and verifying they can be read back correctly.
It is useful for monitoring data integrity in production environments and for post-deployment validation testing.

## Overview

Tempo Vulture performs the following operations:

- Write traces: Pushes test traces to Tempo using OTLP over gRPC
- Read traces by ID: Queries Tempo to retrieve traces by their trace ID
- Search traces: Uses TraceQL to search for traces by attributes
- Metrics queries: Validates TraceQL metrics queries (enabled by default)

Tempo Vulture can run in two modes:

- Continuous mode (default): Runs indefinitely, continuously writing and reading traces while exposing Prometheus metrics
- Validation mode: Executes a fixed number of write/read cycles and exits with a status code indicating success or failure

## Run Tempo Vulture

Tempo Vulture is available as a Docker image:

```bash
docker run grafana/tempo-vulture:latest [arguments...]
```

You can also build it from source:

```bash
make tempo-vulture
./tempo-vulture [arguments...]
```

## Configuration flags

Configure Tempo Vulture using command-line flags.

### Required flags

| Flag                | Description                                                       | Example             |
| ------------------- | ----------------------------------------------------------------- | ------------------- |
| `--tempo-query-url` | The URL (scheme://hostname) at which to query Tempo               | `http://tempo:3200` |
| `--tempo-push-url`  | The URL (scheme://hostname:port) at which to push traces to Tempo | `http://tempo:4317` |

### Optional flags

| Flag                                     | Default    | Description                                                                       |
| ---------------------------------------- | ---------- | --------------------------------------------------------------------------------- |
| `--tempo-org-id`                         | `""`       | The org ID to use when querying Tempo (for multi-tenant deployments)              |
| `--tempo-push-tls`                       | `false`    | Whether to use TLS when pushing spans to Tempo                                    |
| `--tempo-write-backoff-duration`         | `15s`      | Time to pause between write operations                                            |
| `--tempo-long-write-backoff-duration`    | `1m`       | Time to pause between long write operations                                       |
| `--tempo-read-backoff-duration`          | `30s`      | Time to pause between read operations                                             |
| `--tempo-search-backoff-duration`        | `1m`       | Time to pause between search operations. Set to `0s` to disable search validation |
| `--tempo-metrics-backoff-duration`       | `0s`       | Time to pause between TraceQL metrics operations. Set to `0s` to disable          |
| `--tempo-retention-duration`             | `336h`     | The block retention that Tempo is using                                           |
| `--tempo-recent-traces-backoff-duration` | `14m`      | Cutoff between recent and old traces query checks                                 |
| `--tempo-query-livestore`                | `false`    | Whether to query live stores                                                      |
| `--prometheus-listen-address`            | `:80`      | Address to listen on for Prometheus scrapes                                       |
| `--prometheus-path`                      | `/metrics` | Path to publish Prometheus metrics                                                |

### Validation mode flags

| Flag                   | Default | Description                                                       |
| ---------------------- | ------- | ----------------------------------------------------------------- |
| `--validation-mode`    | `false` | Run in validation mode: execute a fixed number of cycles and exit |
| `--validation-cycles`  | `3`     | Number of write/read cycles to perform in validation mode         |
| `--validation-timeout` | `5m`    | Maximum time to run validation mode before timing out             |

## Continuous mode

In continuous mode, Tempo Vulture runs indefinitely and exposes Prometheus metrics that can be used for monitoring and alerting.

### Example

```bash
docker run grafana/tempo-vulture:latest \
  --tempo-query-url=http://tempo:3200 \
  --tempo-push-url=http://tempo:4317 \
  --tempo-org-id=my-tenant \
  --tempo-write-backoff-duration=15s \
  --tempo-read-backoff-duration=30s \
  --tempo-search-backoff-duration=60s
```

### Prometheus metrics

Tempo Vulture exposes the following key metrics:

| Metric                            | Type    | Description                                |
| --------------------------------- | ------- | ------------------------------------------ |
| `tempo_vulture_trace_total`       | Counter | Total number of trace operations performed |
| `tempo_vulture_trace_error_total` | Counter | Total number of trace errors by error type |

Error types include:

- `incorrectresult`: Retrieved trace doesn't match the expected trace
- `incorrect_metrics_result`: Metrics query returned unexpected results
- `missingspans`: Trace has missing spans
- `notfound_byid`: Trace not found by ID
- `notfound_search`: Trace not found via search
- `notfound_traceql`: Trace not found via TraceQL
- `notfound_metrics`: Trace not found via metrics query
- `notfound_search_attribute`: No searchable attribute found in trace
- `inaccurate_metrics`: Metrics count doesn't match actual span count
- `requestfailed`: Request to Tempo failed

### Alerting

You can configure alerts based on Tempo Vulture metrics.
For example, the Tempo mixin includes a [`TempoVultureHighErrorRate` alert](https://github.com/grafana/tempo/blob/08325a7d5da6a330ac5564265eee52e13544abc9/operations/tempo-mixin/alerts.libsonnet#L426) that fires when the error rate exceeds a configurable threshold.

## Validation mode

{{< admonition type="note" >}}
Validation mode was added in Tempo 2.10.
{{< /admonition >}}

Validation mode runs Tempo Vulture as a one-shot test that can be integrated into CI/CD pipelines for post-deployment validation.
Instead of running continuously, it executes a fixed number of write/read cycles and exits with a status code indicating success or failure.

{{< admonition type="warning" >}}
Running Vulture as a Deployment while in validation mode causes problems in Kubernetes. Because Deployments expect long-running processes, a container that finishes its task and exits triggers a `CrashLoopBackOff` (even with exit code `0`). Run validation mode as a Job instead.
{{< /admonition >}}

### Exit codes

| Exit code | Meaning                                                |
| --------- | ------------------------------------------------------ |
| `0`       | All validations passed                                 |
| `1`       | One or more validations failed, or configuration error |

### Environment variables

Validation mode requires authentication:

| Variable                    | Description                                                                     |
| --------------------------- | ------------------------------------------------------------------------------- |
| `TEMPO_ACCESS_POLICY_TOKEN` | Access policy token for authenticating with Tempo (required in validation mode) |

### Example

```bash
docker run \
  -e TEMPO_ACCESS_POLICY_TOKEN=your-token \
  grafana/tempo-vulture:latest \
  --tempo-query-url=https://tempo.example.com \
  --tempo-push-url=https://tempo.example.com:4317 \
  --tempo-org-id=my-tenant \
  --validation-mode \
  --validation-cycles=5 \
  --validation-timeout=10m
```

### CI/CD pipeline integration

Validation mode is useful for:

- **Post-deployment testing**: Verify Tempo is functioning correctly after a deployment
- **Smoke testing**: Quick validation that write and read paths are working
- **Integration testing**: Ensure end-to-end trace pipeline is operational

Example CI/CD usage in a GitHub Actions workflow:

```yaml
- name: Validate Tempo deployment
  run: |
    docker run \
      -e TEMPO_ACCESS_POLICY_TOKEN=${{ secrets.TEMPO_TOKEN }} \
      grafana/tempo-vulture:latest \
      --tempo-query-url=${{ vars.TEMPO_URL }} \
      --tempo-push-url=${{ vars.TEMPO_PUSH_URL }} \
      --tempo-org-id=${{ vars.TEMPO_ORG_ID }} \
      --validation-mode \
      --validation-cycles=3 \
      --validation-timeout=5m
```

### Validation process

In validation mode, Tempo Vulture performs these steps:

1. For each cycle (default: 3):
   - Write a test trace to Tempo
   - Wait briefly for ingestion
   - Read the trace back by ID and verify it matches

2. After all cycles complete:
   - Wait for traces to be searchable (if search is enabled)
   - Search for each trace using a random attribute
   - Verify all traces are found

3. Exit with code `0` if all validations pass, or `1` if any fail

## Docker Compose example

The Tempo repository includes Docker Compose examples that demonstrate how to run Tempo Vulture.
Refer to the [example configurations](https://github.com/grafana/tempo/tree/main/example/docker-compose) for complete setups.

Example from a Docker Compose file:

```yaml
vulture:
  image: grafana/tempo-vulture:latest
  command:
    - "-tempo-push-url=http://distributor:4317"
    - "-tempo-query-url=http://query-frontend:3200"
    - "-tempo-org-id="
```
