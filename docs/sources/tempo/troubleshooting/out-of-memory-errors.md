---
title: Troubleshoot out-of-memory errors
menuTitle: Out-of-memory errors
description: Gain an understanding of how to debug out-of-memory (OOM) errors.
weight: 600
---

# Troubleshoot out-of-memory errors

Learn about out-of-memory (OOM) issues and how to troubleshoot them.

## Set the max attribute size to help control out of memory errors

Tempo queriers can run out of memory when fetching traces that have spans with very large attributes.
This issue has been observed when trying to fetch a single trace using the [`tracebyID` endpoint](https://grafana.com/docs/tempo/latest/api_docs/#query).

To avoid these out-of-memory crashes, use `max_attribute_bytes` to limit the maximum allowable size of any individual attribute.
Any key or values that exceed the configured limit are truncated before storing.

Use the `tempo_distributor_attributes_truncated_total` metric to track how many attributes are truncated.

```yaml
   # Optional
    # Configures the max size an attribute can be. Any key or value that exceeds this limit will be truncated before storing
    # Setting this parameter to '0' would disable this check against attribute size
    [max_attribute_bytes: <int> | default = '2048']
```

Refer to the [configuration for distributors](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#set-max-attribute-size-to-help-control-out-of-memory-errors) documentation for more information.

## Max trace size

Traces which are long-running (minutes or hours) or large (100K - 1M spans) will spike the memory usage of each component when it is encountered.
This is because Tempo treats traces as single units, and keeps all data for a trace together to enable features like structural queries and analysis.

When reading a large trace, it can spike the memory usage of the read components:

* query-frontend
* querier
* ingester
* metrics-generator

When writing a large trace, it can spike the memory usage of the write components:

* ingester
* compactor
* metrics-generator

Start with a smaller trace size limit of 15MB, and increase it as needed.
With an average span size of 300 bytes, this allows for 50K spans per trace.

Always ensure that the limit is configured, and the largest recommended limit is 60 MB.

Configure the limit in the per-tenant overrides:

```yaml
overrides:
    'tenant123':
        max_bytes_per_trace: 1.5e+07
```

Refer to the [Overrides](# https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#standard-overrides) documentation for more information.

## Large attributes

Very large attributes, 10KB or longer, can spike the memory usage of each component when they are encountered.
Tempo's Parquet format uses dictionary-encoded columns, which works well for repeated values.
However, for very large and high cardinality attributes, this can require a large amount of memory.

A common source of large attributes is auto-instrumentation in these areas:

* HTTP
    * Request or response bodies
    * Large headers
        * [http.request.header.&lt;key>](https://opentelemetry.io/docs/specs/semconv/attributes-registry/http/)
    * Large URLs
        * http.url
        * [url.full](https://opentelemetry.io/docs/specs/semconv/attributes-registry/url/)
* Databases
    * Full query statements
    * db.statement
    * [db.query.text](https://opentelemetry.io/docs/specs/semconv/attributes-registry/db/)
* Queues
    * Message bodies

When reading these attributes, they can spike the memory usage of the read components:

* query-frontend
* querier
* ingester
* metrics-generator

When writing these attributes, they can spike the memory usage of the write components:
* ingester
* compactor
* metrics-generator

You can [automatically limit attribute sizes](https://github.com/grafana/tempo/pull/4335) using [`max_attribute_bytes`]((https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#set-max-attribute-size-to-help-control-out-of-memory-errors).
You can also use these options:

* Manually update application instrumentation to remove or limit these attributes
* Drop the attributes in the tracing pipeline using [attribute processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/attributesprocessor)
