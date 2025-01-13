---
title: Troubleshoot out-of-memory errors
menuTitle: Out-of-memory errors
description: Gain an understanding of how to debug out-of-memory (OOM) errors.
weight: 600
---

# Troubleshoot out-of-memory errors

Learn about out-of-memory (OOM) errors and how to troubleshoot them.

## Set the max attribute size to help control out of memory errors

Tempo queriers can run out of memory when fetching traces that have spans with very large attributes.
This issue has been observed when trying to fetch a single trace using the [`tracebyID` endpoint](https://grafana.com/docs/tempo/latest/api_docs/#query).

While a trace might not have a lot of spans (roughly 500), it can have a larger size (approximately 250KB).
Some of the spans in that trace had attributes whose values were very large in size.

To avoid these out-of-memory crashes, use `max_span_attr_byte` to limit the maximum allowable size of any individual attribute.
Any key or values that exceed the configured limit are truncated before storing.
The default value is `2048`.

```yaml
   # Optional
    # Configures the max size an attribute can be. Any key or value that exceeds this limit will be truncated before storing
    # Setting this parameter to '0' would disable this check against attribute size
    [max_span_attr_byte: <int> | default = '2048']
```

Refer to the [configuration for distributors](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#distributor) documentation.