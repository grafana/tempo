---
aliases:
  - /docs/tempo/latest/metrics-generator/multitenancy
title: Multitenancy support
description: Learn about multitenancy support in the metrics-generator.
weight: 600
aliases:
  - ../../metrics-generator/multitenancy/ # /docs/tempo/<TEMPO_VERSION>/metrics-generator/multitenancy/
---

# Multitenancy support

Multitenancy is supported in the metrics-generator through the use of environment variables and per-tenant overrides.
This is useful when you want to propagate the multitenancy to the metrics backend,
keeping the data separated and secure.

## Requirements

- Tempo version 2.4.0 or later

## Usage

To use this feature, you need to define the `remote_write_headers` override for each tenant in your configuration.
You can also use environment variables in your configuration file, which will be expanded at runtime.
To make use of environment variables, you need to pass the `--config.expand-env` flag to Tempo.

Example:

```yaml
overrides:
  team-traces-a:
    metrics_generator:
      processors: [ 'span-metrics' ]
      remote_write_headers:
        Authorization: ${PROM_A_BASIC_AUTH}
  team-traces-b:
    metrics_generator:
      processors: [ 'span-metrics', 'service-graphs' ]
      remote_write_headers:
        Authorization: ${PROM_B_BEARER_AUTH}
```

```bash
export PROM_A_BASIC_AUTH="Basic $(echo "team-a:$(cat /token-prometheus-a)"|base64|tr -d '[:space:]')"
export PROM_B_BEARER_AUTH="Bearer $(cat /token-prometheus-b)"
```

In this example, `PROM_A_BASIC_AUTH` and `PROM_B_BEARER_AUTH` are environment variables that contain the respective tenants' authorization tokens.
The `remote_write_headers` override is used to specify the `Authorization` header for each tenant.
The `Authorization` header is used to authenticate the remote write request to the Prometheus remote write endpoint.