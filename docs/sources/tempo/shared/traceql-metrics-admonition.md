---
headless: true
description: Shared file for TraceQL metrics admonition.
labels:
  products:
    - enterprise
    - oss
---

[//]: # 'This file creates a caution admonition for TraceQL metrics.'
[//]: # 'This shared file is included in these locations:'
[//]: # '/tempo/docs/sources/tempo/traceql/metrics-queries/_index.md'
[//]: # '/tempo/docs/sources/tempo/traceql/metrics-queries/traceql-metrics-admonition.md'
[//]: # '/tempo/docs/sources/tempo/traceql/_index.md'
[//]: #
[//]: # 'If you make changes to this file, verify that the meaning and content are not changed in any place where the file is included.'
[//]: # 'Any links should be fully qualified and not relative.'

<!-- Using a custom admonition because no feature flag is required. -->
{{< admonition type="caution" >}}
TraceQL metrics is an [experimental feature](/docs/release-life-cycle/). Engineering and on-call support is not available. Documentation is either limited or not provided outside of code comments. No SLA is provided.
Contact Grafana Support to enable this feature in Grafana Cloud.
{{< /admonition >}}