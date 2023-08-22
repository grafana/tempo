# Filter Processor

| Status                   |                       |
| ------------------------ | --------------------- |
| Stability                | [alpha]               |
| Supported pipeline types | metrics, logs, traces |
| Distributions            | [core], [contrib]     |

The filter processor can be configured to include or exclude:

- Logs, based on OTTL conditions or resource attributes using the `strict` or `regexp` match types
- Metrics based on OTTL Conditions or metric name in the case of the `strict` or `regexp` match types,
  or based on other metric attributes in the case of the `expr` match type.
  Please refer to [config.go](./config.go) for the config spec.
- Data points based on OTTL conditions
- Spans based on OTTL conditions or span names and resource attributes, all with full regex support
- Span Events based on OTTL conditions.

For OTTL conditions configuration see [OTTL](#ottl).  For all other options, continue reading.

It takes a pipeline type, of which `logs` `metrics`, and `traces` are supported, followed
by an action:

- `include`: Any names NOT matching filters are excluded from remainder of pipeline
- `exclude`: Any names matching filters are excluded from remainder of pipeline

For the actions the following parameters are required:

For logs:

- `match_type`: `strict`|`regexp`
- `resource_attributes`: ResourceAttributes defines a list of possible resource
  attributes to match logs against.
  A match occurs if any resource attribute matches all expressions in this given list.
- `record_attributes`: RecordAttributes defines a list of possible record
  attributes to match logs against.
  A match occurs if any record attribute matches all expressions in this given list.
- `severity_texts`: SeverityTexts defines a list of possible severity texts to match the logs against.
  A match occurs if the record matches any expression in this given list.
- `bodies`: Bodies defines a list of possible log bodies to match the logs against.
  A match occurs if the record matches any expression in this given list.
- `severity_number`: SeverityNumber defines how to match a record based on its SeverityNumber.
  The following can be configured for matching a log record's SeverityNumber:
  - `min`: Min defines the minimum severity with which a log record should match.
    e.g. if this is "WARN", all log records with "WARN" severity and above (WARN[2-4], ERROR[2-4], FATAL[2-4]) are matched.
    The list of valid severities that may be used for this option can be found [here](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/data-model.md#displaying-severity). You may use either the numerical "SeverityNumber" or the "Short Name"
  - `match_undefined`: MatchUndefinedSeverity defines whether to match logs with undefined severity or not when using the `min_severity` matching option.
    By default, this is `false`.

For metrics:

- `match_type`: `strict`|`regexp`|`expr`
- `metric_names`: (only for a `match_type` of `strict` or `regexp`) list of strings
  or re2 regex patterns
- `expressions`: (only for a `match_type` of `expr`) list of `expr` expressions
  (see "Using an `expr` match_type" below)
- `resource_attributes`: ResourceAttributes defines a list of possible resource
  attributes to match metrics against.
  A match occurs if any resource attribute matches all expressions in this given list.

This processor uses [re2 regex][re2_regex] for regex syntax.

[re2_regex]: https://github.com/google/re2/wiki/Syntax

More details can be found at [include/exclude metrics](../attributesprocessor/README.md#includeexclude-filtering).

Examples:

```yaml
processors:
  filter/1:
    metrics:
      include:
        match_type: regexp
        metric_names:
          - prefix/.*
          - prefix_.*
        resource_attributes:
          - key: container.name
            value: app_container_1
      exclude:
        match_type: strict
        metric_names:
          - hello_world
          - hello/world
  filter/2:
    logs:
      include:
        match_type: strict
        resource_attributes:
          - key: host.name
            value: just_this_one_hostname
  filter/regexp:
    logs:
      include:
        match_type: regexp
        resource_attributes:
          - key: host.name
            value: prefix.*
  filter/regexp_record:
    logs:
      include:
        match_type: regexp
        record_attributes:
          - key: record_attr
            value: prefix_.*
  # Filter on severity text field
  filter/severity_text:
    logs:
      include:
        match_type: regexp
        severity_texts:
        - INFO[2-4]?
        - WARN[2-4]?
        - ERROR[2-4]?
  # Filter out logs below INFO (no DEBUG or TRACE level logs),
  # retaining logs with undefined severity
  filter/severity_number:
    logs:
      include:
        severity_number:
          min: "INFO"
          match_undefined: true
  filter/bodies:
    logs:
      include:
        match_type: regexp
        bodies:
        - ^IMPORTANT RECORD
```

Refer to the config files in [testdata](./testdata) for detailed
examples on using the processor.

## Using an "expr" match_type

In addition to matching metric names with the `strict` or `regexp` match types, the filter processor
supports matching entire `Metric`s using the [expr](https://github.com/antonmedv/expr) expression engine.

The `expr` filter evaluates the supplied boolean expressions _per datapoint_ on a metric, and returns a result
for the entire metric. If any datapoint evaluates to true then the entire metric evaluates to true, otherwise
false.

Made available to the expression environment are the following:

* `MetricName`
    a variable containing the current Metric's name
* `MetricType`
    a variable containing the current Metric's type: "Gauge", "Sum", "Histogram", "ExponentialHistogram" or "Summary".
* `Label(name)`
    a function that takes a label name string as an argument and returns a string: the value of a label with that
    name if one exists, or ""
* `HasLabel(name)`
    a function that takes a label name string as an argument and returns a boolean: true if the datapoint has a label
    with that name, false otherwise

Example:

```yaml
processors:
  filter/1:
    metrics:
      exclude:
        match_type: expr
        expressions:
        - MetricName == "my.metric" && Label("my_label") == "abc123"
        - MetricType == "Histogram"
```

The above config will filter out any Metric that both has the name "my.metric" and has at least one datapoint
with a label of 'my_label="abc123"'.

### Support for multiple expressions

As with `strict` and `regexp`, multiple `expr` expressions are allowed.

For example, the following two filters have the same effect: they filter out metrics named "system.cpu.time" and
"system.disk.io". 

```yaml
processors:
  filter/expr:
    metrics:
      exclude:
        match_type: expr
        expressions:
          - MetricName == "system.cpu.time"
          - MetricName == "system.disk.io"
  filter/strict:
    metrics:
      exclude:
        match_type: strict
        metric_names:
          - system.cpu.time
          - system.disk.io
```

The expressions are effectively ORed per datapoint. So for the above `expr` configuration, given a datapoint, if its
parent Metric's name is "system.cpu.time" or "system.disk.io" then there's a match. The conditions are tested against
all the datapoints in a Metric until there's a match, in which case the entire Metric is considered a match, and in
the above example the Metric will be excluded. If after testing all the datapoints in a Metric against all the
expressions there isn't a match, the entire Metric is considered to be not matching.


### Filter metrics using resource attributes
In addition to the names, metrics can be filtered using resource attributes. `resource_attributes` takes a list of resource attributes to filter metrics against. 

Following example will include only the metrics coming from `app_container_1` (the value for `container.name` resource attribute is `app_container_1`). 

```yaml
processors:
  filter/resource_attributes_include:
    metrics:
      include:
        match_type: strict
        metric_names:
          - hello_world
          - hello/world
        resource_attributes:
          - key: container.name
            value: app_container_1
```

Following example will exclude all the metrics coming from `app_container_1` (the value for `container.name` resource attribute is `app_container_1`). 

```yaml
processors:
  filter/resource_attributes_exclude:
    metrics:
      exclude:
        match_type: strict
        metric_names:
          - hello_world
          - hello/world
        resource_attributes:
          - key: container.name
            value: app_container_1
```

We can also use `regexp` to filter metrics using resource attributes. Following example will include only the metrics coming from `app_container_1` or `app_container_2` (the value for `container.name` resource attribute is either `app_container_1` or `app_container_2`). 

```yaml
processors:
  filter/resource_attributes_regexp:
    metrics:
      exclude:
        match_type: regexp
        metric_names:
          - hello_world
          - hello/world
        resource_attributes:
          - key: container.name
            value: (app_container_1|app_container_1)
```

In case the no metric names are provided, `matric_names` being empty, the filtering is only done at resource level.

### Filter Spans from Traces

* This pipeline is able to drop spans and whole traces 
* Note: If this drops a parent span, it does not search out it's children leading to a missing Span in your trace visualization

See the documentation in the [attribute processor](../attributesprocessor/README.md) for syntax

For spans, one of Services, SpanNames, Attributes, Resources or Libraries must be specified with a
non-empty value for a valid configuration.

```yaml
processors:
  filter/spans:
    spans:
      include:
        match_type: strict
        services:
          - app_3
      exclude:
        match_type: regexp
        services:
          - app_1
          - app_2
        span_names:
          - hello_world
          - hello/world
        attributes:
          - key: container.name
            value: (app_container_1|app_container_2)
        libraries:
          - name: opentelemetry
            version: 0.0-beta
        resources:
          - key: container.host
            value: (localhost|127.0.0.1)
```

## OTTL
The [OpenTelemetry Transformation Language](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/README.md) is a language for interacting with telemetry within the collector in generic ways.
The filterprocessor can be configured to use OTTL conditions to determine when to drop telemetry.
If any condition is met, the telemetry is dropped (each condition is ORed together).
Each configuration option corresponds with a different type of telemetry and OTTL Context.
See the table below for details on each context and the fields it exposes.

| Config              | OTTL Context                                                                                                                       |
|---------------------|------------------------------------------------------------------------------------------------------------------------------------|
| `traces.span`       | [Span](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/contexts/ottlspan/README.md)           |
| `traces.spanevent`  | [SpanEvent](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/contexts/ottlspanevent/README.md) |
| `metrics.metric`    | [Metric](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/contexts/ottlmetric/README.md)       |
| `metrics.datapoint` | [DataPoint](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/contexts/ottldatapoint/README.md) |
| `logs.log_record`   | [Log](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/contexts/ottllog/README.md)             |

The OTTL allows the use of `and`, `or`, and `()` in conditions.
See [OTTL Boolean Expressions](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/README.md#boolean-expressions) for more details.

For conditions that apply to the same signal, such as spans and span events, if the "higher" level telemetry matches a condition and is dropped, the "lower" level condition will not be checked.
This means that if a span is dropped but a span event condition was defined, the span event condition will not be checked.
The same relationship applies to metrics and datapoints.

If all span events for a span are dropped, the span will be left intact.
If all datapoints for a metric are dropped, the metric will also be dropped.

The filter processor also allows configuring an optional field, `error_mode`, which will determine how the processor reacts to errors that occur while processing an OTTL condition.

| error_mode            | description                                                                                                                |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------|
| ignore                | The processor ignores errors returned by conditions and continues on to the next condition.  This is the recommended mode. |
| propagate             | The processor returns the error up the pipeline.  This will result in the payload being dropped from the collector.        |

If not specified, `propagate` will be used.

### OTTL Functions

The filter processor has access to all the [factory functions of the OTTL](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/ottlfuncs#ottl-functions)

In addition, the processor defines a few of its own functions:

**Metrics only functions**
- [HasAttrKeyOnDatapoint](#HasAttrKeyOnDatapoint)
- [HasAttrOnDatapoint](#HasAttrOnDatapoint)

#### HasAttrKeyOnDatapoint

`HasAttrKeyOnDatapoint(key)`

Returns `true` if the given key appears in the attribute map of any datapoint on a metric.
`key` must be a string.

Examples:

- `HasAttrKeyOnDatapoint("http.method")`

#### HasAttrOnDatapoint

`HasAttrOnDatapoint(key, value)`

Returns `true` if the given key and value appears in the attribute map of any datapoint on a metric.
`key` and `value` must both be strings.

Examples:

- `HasAttrOnDatapoint("http.method", "GET")`

### OTTL Examples

```yaml
processors:
  filter/ottl:
    error_mode: ignore
    traces:
      span:
        - 'attributes["container.name"] == "app_container_1"'
        - 'resource.attributes["host.name"] == "localhost"'
        - 'name == "app_3"'
      spanevent:
        - 'attributes["grpc"] == true'
        - 'IsMatch(name, ".*grpc.*") == true'
    metrics:
      metric:
          - 'name == "my.metric" and resource.attributes["my_label"] == "abc123"'
          - 'type == METRIC_DATA_TYPE_HISTOGRAM'
      datapoint:
          - 'metric.type == METRIC_DATA_TYPE_SUMMARY'
          - 'resource.attributes["service.name"] == "my_service_name"'
    logs:
      log_record:
        - 'IsMatch(body, ".*password.*") == true'
        - 'severity_number < SEVERITY_NUMBER_WARN'
```

[alpha]:https://github.com/open-telemetry/opentelemetry-collector#alpha
[contrib]:https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
[core]:https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol
