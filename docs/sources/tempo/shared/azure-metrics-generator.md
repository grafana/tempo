---
headless: true
description: Azure metrics generator configuration.
labels:
  products:
    - enterprise
    - oss
---

[//]: # "This file documents the Azure metrics generator configuration for Tempo."
[//]: # "This shared file is included in these locations:"
[//]: # "/tempo/docs/sources/tempo/configuration/hosted-storage/azure.md"
[//]: #
[//]: # "If you make changes to this file, verify that the meaning and content are not changed in any place where the file is included."
[//]: # "Any links should be fully qualified and not relative: /docs/grafana/ instead of ../grafana/."

<!-- local blocks processor, Azure storage, and metrics-generator -->

By default the metrics-generator doesn't require a backend connection unless you've enabled the local-blocks processor.
When this configuration is set, the metrics-generator produces blocks and flushes them into a backend storage.

In this case, list the generator in the `env var` expansion configuration so the `STORAGE_ACCOUNT_ACCESS_KEY` has the secret value.

```yaml
generator:
  extraArgs:
    - "-config.expand-env=true"
  extraEnv:
    - name: STORAGE_ACCOUNT_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: tempo-traces-stg-key
          key: tempo-traces-key
```
