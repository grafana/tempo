---
headless: true
description: Azure metrics generator configuration for Helm charts.
labels:
  products:
    - enterprise
    - oss
---

[//]: # "This file documents the Azure metrics generator configuration for Tempo when using Helm charts."
[//]: # "This shared file is included in these locations:"
[//]: # "/tempo/docs/sources/tempo/configuration/hosted-storage/azure.md"
[//]: # "/tempo/docs/sources/tempo/metrics-from-traces/metrics-queries/configure-traceql-metrics.md"
[//]: # "/helm-charts/tempo-distributed/get-started-helm-charts/_index.md"
[//]: #
[//]: # "If you make changes to this file, verify that the meaning and content are not changed in any place where the file is included."
[//]: # "Any links should be fully qualified and not relative: /docs/grafana/ instead of ../grafana/."

<!-- metrics-generator, Azure storage, and Helm charts-->

The metrics-generator consumes trace data from Kafka and writes derived metrics using the Prometheus remote write protocol.
If the metrics-generator needs access to Azure blob storage, list it in the `env var` expansion configuration so the `STORAGE_ACCOUNT_ACCESS_KEY` has the secret value.

You can use this configuration example with Helm charts, like `tempo-distributed`.
Replace any values in all caps with the values for your Helm deployment.

```yaml
generator:
  extraArgs:
    - "-config.expand-env=true"
  extraEnv:
    - name: <STORAGE_ACCOUNT_ACCESS_KEY>
      valueFrom:
        secretKeyRef:
          name: <TEMPO-TRACES-STG-KEY>
          key: <TEMPO-TRACES-KEY>
```
