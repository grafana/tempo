---
title: Apache Parquet schema
menuTitle: Parquet schema
description: This document describes the schema used with the Parquet block format.
weight: 900
aliases:
  - /docs/tempo/parquet/schema
---

# Apache Parquet schema

<!-- vale Grafana.GoogleSpacing = NO -->
<!-- vale Grafana.We = NO -->
<!-- vale Grafana.GooglePassive = NO -->
<!-- vale Grafana.GoogleWill = NO -->

Starting with Tempo 2.0, Apache Parquet is used as the default column-formatted block format.
Refer to the [Parquet configuration options](../../configuration/parquet/) for more information.

This document describes the schema used with the Parquet block format.

## Version applicability
 
Tempo 2.10 defaults to the vParquet4 schema. vParquet5 is production-ready and differs in some schema details.
Unless otherwise noted, the sections below describe vParquet4.

The following sections apply to both vParquet4 and vParquet5:

- Fully nested versus span-oriented schema
- Static vs dynamic columns (see vParquet5 differences for changes to dedicated columns)
- Compression and encoding
- Bloom filters

## Fully nested versus span-oriented schema

There are two overall approaches to a columnar schema: fully nested or span-oriented.
Span-oriented means a flattened schema where traces are destructured into rows of spans.
A fully nested schema means the current trace structures such as Resource/Scope/Spans/Events are preserved (nested data is natively supported in Parquet).
In both cases, individual leaf values such as span name and duration are individual columns.

We chose the nested schema for several reasons:

- The block size is much smaller for the nested schema. This is due to the high data duplication incurred when flattening resource-level attributes such as `service.name` to each individual span.
- A flat schema is not truly "flat" because each span still contains nested data such as attributes and events.
- Nested schema is much faster to search for resource-level attributes because the resource-level columns are very small (1 row for each batch).
- Translation to and from the OpenTelemetry Protocol Specification (OTLP) is straightforward.
- Easily add computed columns (for example, trace duration) at multiple levels such as per-trace, per-batch, etc.

## Static vs dynamic columns

Dynamic vs static columns add another layer to the schema.
A dynamic schema stores each attribute such as `service.name` and `http.status_code` as its own column and the columns in each parquet file can be different.
A static schema is unresponsive to the shape of the data, and all attributes are stored in generic key/value containers.

The dynamic schema is the ultimate dream for a columnar format but it is too complex for a first release.
However, the benefits of that approach are also too good to pass up, so we propose a hybrid approach.
It is primarily a static schema but with some dynamic columns extracted from trace data based on some heuristics of frequently queried attributes.
We plan to continue investing in this direction to implement a fully dynamic schema where trace attributes are blown out into independent Parquet columns at runtime.

For more information, refer to the [Parquet design document](https://github.com/grafana/tempo/blob/main/docs/design-proposals/2022-04%20Parquet.md).

## Schema details

The adopted Parquet schema is mostly a direct translation of OTLP but with some key differences.

The table below uses these abbreviations:

- `rs` - resource spans
- `ss` - scope spans

<!-- vale Grafana.GoogleSpacing = NO -->

|                                                       |            |                                                                                                                                                                                                                                                                                               |
| :---------------------------------------------------- | :--------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Name                                                  | Type       | Description                                                                                                                                                                                                                                                                                   |
| TraceID                                               | byte array | The trace ID in 16-byte binary form.                                                                                                                                                                                                                                                          |
| TraceIDText                                           | string     | The trace ID in hexadecimal text form.                                                                                                                                                                                                                                                        |
| StartTimeUnixNano                                     | int64      | Start time of the first span in the trace, in nanoseconds since unix epoch.                                                                                                                                                                                                                   |
| EndTimeUnixNano                                       | int64      | End time of the last span in the trace, in nanoseconds since unix epoch.                                                                                                                                                                                                                      |
| DurationNano                                          | int64      | Total trace duration in nanoseconds, computed as difference between EndTimeUnixNano and StartTimeUnixNano.                                                                                                                                                                                    |
| RootServiceName                                       | string     | The resource-level `service.name` attribute (rs.Resource.ServiceName) from the root span of the trace if one exists, else empty string.                                                                                                                                                       |
| RootSpanName                                          | string     | The name (rs.ss.Spans.Name) of the root span if one exists, else empty string.                                                                                                                                                                                                                |
| ServiceStats                                          | map        | Per-service counts keyed by service name. Values include span count and error count.                                                                                                                                                                                                          |
| rs                                                    |            | Short-hand for ResourceSpans                                                                                                                                                                                                                                                                  |
| rs.Resource.ServiceName                               | string     | A dedicated column for the resource-level `service.name` attribute if present. https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/#service                                                                                                                   |
| rs.Resource.Cluster                                   | string     | A dedicated column for the resource-level `cluster` attribute if present and of string type. Values of other types will be stored in the generic attribute columns.                                                                                                                           |
| rs.Resource.Namespace                                 | string     | A dedicated column for the resource-level `namespace` attribute if present and of string type. Values of other types will be stored in the generic attribute columns.                                                                                                                         |
| rs.Resource.Pod                                       | string     | A dedicated column for the resource-level `pod` attribute if present and of string type. Values of other types will be stored in the generic attribute columns.                                                                                                                               |
| rs.Resource.Container                                 | string     | A dedicated column for the resource-level `container` attribute if present and of string type. Values of other types will be stored in the generic attribute columns.                                                                                                                         |
| rs.Resource.K8sClusterName                            | string     | A dedicated column for the resource-level `k8s.cluster.name` attribute if present and of string type. Values of other types will be stored in the generic attribute columns. https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/k8s/#cluster                 |
| rs.Resource.K8sNamespaceName                          | string     | A dedicated column for the resource-level `k8s.namespace.name` attribute if present and of string type. Values of other types will be stored in the generic attribute columns. https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/k8s/#namespace             |
| rs.Resource.K8sPodName                                | string     | A dedicated column for the resource-level `k8s.pod.name` attribute if present and of string type. Values of other types will be stored in the generic attribute columns. https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/k8s/#pod                         |
| rs.Resource.K8sContainerName                          | string     | A dedicated column for the resource-level `k8s.container.name` attribute if present and of string type. Values of other types will be stored in the generic attribute columns. https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/k8s/#container             |
| rs.Resource.DroppedAttributesCount                    | int        | Number of resource attributes that were dropped.                                                                                                                                                                                                                                              |
| rs.Resource.Attrs.Key                                 | string     | All resource attributes that do not have a dedicated column are stored as a key value pair in these columns. The Key column stores the name, and then one of the Value columns is populated according to the attribute's data type. The other value columns will contain null.                |
| rs.Resource.Attrs.IsArray                             | bool       | Indicates if the attribute is stored as an array.                                                                                                                                                                                                                                             |
| rs.Resource.Attrs.Value                               | string     | The attribute value if string type (or array of strings), else null.                                                                                                                                                                                                                          |
| rs.Resource.Attrs.ValueInt                            | int        | The attribute value if integer type (or array of integers), else null.                                                                                                                                                                                                                        |
| rs.Resource.Attrs.ValueDouble                         | float      | The attribute value if float type (or array of floats), else null.                                                                                                                                                                                                                            |
| rs.Resource.Attrs.ValueBool                           | bool       | The attribute value if boolean type (or array of booleans), else null.                                                                                                                                                                                                                        |
| rs.Resource.Attrs.ValueUnsupported                    | string     | JSON-encoded AnyValue for unsupported or mixed-type values.                                                                                                                                                                                                                                   |
| rs.Resource.DedicatedAttributes                       |            | Group containing spares for dedicated attribute columns with resource scope.                                                                                                                                                                                                                  |
| rs.Resource.DedicatedAttributes.String01 ... String10 | string     | 10 spare columns for dedicated attribute columns.                                                                                                                                                                                                                                             |
| rs.ss                                                 |            | Shorthand for ResourceSpans.ScopeSpans                                                                                                                                                                                                                                                        |
| rs.ss.Scope                                           |            | Shorthand for ResourceSpans.ScopeSpans.Scope                                                                                                                                                                                                                                                  |
| rs.ss.Scope.Name                                      | string     | Scope name if present, else empty string. https://opentelemetry.io/docs/specs/otel/glossary/#instrumentation-scope                                                                                                                                                                            |
| rs.ss.Scope.Version                                   | string     | The Scope version if present, else empty string. https://opentelemetry.io/docs/specs/otel/glossary/#instrumentation-scope                                                                                                                                                                     |
| rs.ss.Scope.Attrs                                     |            | Scope attributes, using the same columns as rs.Resource.Attrs.\*                                                                                                                                                                                                                              |
| rs.ss.Scope.DroppedAttributesCount                    | int        | Number of scope attributes that were dropped.                                                                                                                                                                                                                                                 |
| rs.ss.Spans.SpanID                                    | byte array | Span unique ID.                                                                                                                                                                                                                                                                               |
| rs.ss.Spans.ParentSpanID                              | byte array | The unique ID of the span's parent. For root spans without a parent this is null.                                                                                                                                                                                                             |
| rs.ss.Spans.ParentID                                  | int32      | Trace local numeric parent ID.                                                                                                                                                                                                                                                                |
| rs.ss.Spans.NestedSetLeft                             | int32      | Left bound of the nested set model. Also used as a trace local numeric span ID.                                                                                                                                                                                                               |
| rs.ss.Spans.NestedSetRight                            | int32      | Right bound of the nested set model.                                                                                                                                                                                                                                                          |
| rs.ss.Spans.Name                                      | string     | Span name.                                                                                                                                                                                                                                                                                    |
| rs.ss.Spans.StartTimeUnixNano                         | int64      | Start time the span in nanoseconds since unix epoch.                                                                                                                                                                                                                                          |
| rs.ss.Spans.DurationNano                              | int64      | Span duration in nanoseconds.                                                                                                                                                                                                                                                                 |
| rs.ss.Spans.Kind                                      | int        | The span's kind. Defined values: 0. Unset; 1. Internal; 2. Server; 3. Client; 4. Producer; 5. Consumer; https://opentelemetry.io/docs/reference/specification/trace/api/#spankind                                                                                                             |
| rs.ss.Spans.StatusCode                                | int        | The span status. Defined values: 0: Unset; 1: OK; 2: Error. https://opentelemetry.io/docs/reference/specification/trace/api/#set-status                                                                                                                                                       |
| rs.ss.Spans.StatusMessage                             | string     | Optional message to accompany Error status.                                                                                                                                                                                                                                                   |
| rs.ss.Spans.HttpMethod                                | string     | A dedicated column for the span-level `http.method` attribute if present and of string type, else null. Values of other types will be stored in the generic attribute columns. https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/http/#common-attributes       |
| rs.ss.Spans.HttpStatusCode                            | int        | A dedicated column for the span-level `http.status_code` attribute if present and of integer type, else null. Values of other types will be stored in the generic attribute columns. https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/http/#common-attributes |
| rs.ss.Spans.HttpUrl                                   | string     | A dedicated column for the span-level `http.url` attribute if present and of string type, else null. Values of other types will be stored in the generic attribute columns. https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/http/#http-client                |
| rs.ss.Spans.DroppedAttributesCount                    | int        | Number of attributes that were dropped                                                                                                                                                                                                                                                        |
| rs.ss.Spans.Attrs                                     |            | Span attributes, using the same columns as rs.Resource.Attrs.\*                                                                                                                                                                                                                               |
| rs.ss.Spans.DedicatedAttributes                       |            | Group containing spares for dedicated attribute columns with span scope                                                                                                                                                                                                                       |
| rs.ss.Spans.DedicatedAttributes.String01 ... String10 | string     | 10 spare columns used for dedicated attributes                                                                                                                                                                                                                                                |
| rs.ss.Spans.DroppedEventsCount                        | int        | The number of events that were dropped                                                                                                                                                                                                                                                        |
| rs.ss.Spans.Events.TimeSinceStartNano                 | int64      | The event timestamp in nanoseconds, relative to the span start time.                                                                                                                                                                                                                          |
| rs.ss.Spans.Events.Name                               | string     | The event name or message.                                                                                                                                                                                                                                                                    |
| rs.ss.Spans.Events.DroppedAttributesCount             | int        | The number of event attributes that were dropped.                                                                                                                                                                                                                                             |
| rs.ss.Spans.Events.Attrs                              |            | Event attributes, using the same columns as rs.Resource.Attrs.\*                                                                                                                                                                                                                              |
| rs.ss.Spans.DroppedLinksCount                         | int        | The number of links that were dropped.                                                                                                                                                                                                                                                        |
| rs.ss.Spans.Links                                     |            | Repeated link records with TraceID, SpanID, TraceState, Attrs, and DroppedAttributesCount.                                                                                                                                                                                                    |
| rs.ss.Spans.TraceState                                | string     | The span's TraceState value if present, else empty string. https://opentelemetry.io/docs/reference/specification/trace/api/#tracestate                                                                                                                                                        |

<!-- vale Grafana.GoogleSpacing = YES -->

To increase the readability the table omits the groups `list.element` that are added for nested list types in Parquet.
For maps (for example, ServiceStats), Parquet also inserts map-specific group levels that are omitted here.

{{< collapse title="vParquet4 block schema example" >}}

```
message Trace {
  required binary TraceID;
  required binary TraceIDText (STRING);
  required int64 StartTimeUnixNano (INTEGER(64,false));
  required int64 EndTimeUnixNano (INTEGER(64,false));
  required int64 DurationNano (INTEGER(64,false));
  required binary RootServiceName (STRING);
  required binary RootSpanName (STRING);
  optional group ServiceStats (MAP) {
    repeated group key_value {
      required binary key (STRING);
      required group value {
        required int32 SpanCount (INTEGER(32,false));
        required int32 ErrorCount (INTEGER(32,false));
      }
    }
  }
  required group rs (LIST) {
    repeated group list {
      required group element {
        required group Resource {
          required group Attrs (LIST) {
            repeated group list {
              required group element {
                required binary Key (STRING);
                required boolean IsArray;
                required group Value (LIST) {
                  repeated group list {
                    required binary element (STRING);
                  }
                }
                required group ValueInt (LIST) {
                  repeated group list {
                    required int64 element (INTEGER(64,true));
                  }
                }
                required group ValueDouble (LIST) {
                  repeated group list {
                    required double element;
                  }
                }
                required group ValueBool (LIST) {
                  repeated group list {
                    required boolean element;
                  }
                }
                optional binary ValueUnsupported (STRING);
              }
            }
          }
          required int32 DroppedAttributesCount (INTEGER(32,true));
          required binary ServiceName (STRING);
          optional binary Cluster (STRING);
          optional binary Namespace (STRING);
          optional binary Pod (STRING);
          optional binary Container (STRING);
          optional binary K8sClusterName (STRING);
          optional binary K8sNamespaceName (STRING);
          optional binary K8sPodName (STRING);
          optional binary K8sContainerName (STRING);
          required group DedicatedAttributes {
            optional binary String01 (STRING);
            optional binary String02 (STRING);
            optional binary String03 (STRING);
            optional binary String04 (STRING);
            optional binary String05 (STRING);
            optional binary String06 (STRING);
            optional binary String07 (STRING);
            optional binary String08 (STRING);
            optional binary String09 (STRING);
            optional binary String10 (STRING);
          }
        }
        required group ss (LIST) {
          repeated group list {
            required group element {
              required group Scope {
                required binary Name (STRING);
                required binary Version (STRING);
                required group Attrs (LIST) {
                  repeated group list {
                    required group element {
                      required binary Key (STRING);
                      required boolean IsArray;
                      required group Value (LIST) {
                        repeated group list {
                          required binary element (STRING);
                        }
                      }
                      required group ValueInt (LIST) {
                        repeated group list {
                          required int64 element (INTEGER(64,true));
                        }
                      }
                      required group ValueDouble (LIST) {
                        repeated group list {
                          required double element;
                        }
                      }
                      required group ValueBool (LIST) {
                        repeated group list {
                          required boolean element;
                        }
                      }
                      optional binary ValueUnsupported (STRING);
                    }
                  }
                }
                required int32 DroppedAttributesCount (INTEGER(32,true));
              }
              required group Spans (LIST) {
                repeated group list {
                  required group element {
                    required binary SpanID;
                    required binary ParentSpanID;
                    required int32 ParentID (INTEGER(32,true));
                    required int32 NestedSetLeft (INTEGER(32,true));
                    required int32 NestedSetRight (INTEGER(32,true));
                    required binary Name (STRING);
                    required int64 Kind (INTEGER(64,true));
                    required binary TraceState (STRING);
                    required int64 StartTimeUnixNano (INTEGER(64,false));
                    required int64 DurationNano (INTEGER(64,false));
                    required int64 StatusCode (INTEGER(64,true));
                    required binary StatusMessage (STRING);
                    required group Attrs (LIST) {
                      repeated group list {
                        required group element {
                          required binary Key (STRING);
                          required boolean IsArray;
                          required group Value (LIST) {
                            repeated group list {
                              required binary element (STRING);
                            }
                          }
                          required group ValueInt (LIST) {
                            repeated group list {
                              required int64 element (INTEGER(64,true));
                            }
                          }
                          required group ValueDouble (LIST) {
                            repeated group list {
                              required double element;
                            }
                          }
                          required group ValueBool (LIST) {
                            repeated group list {
                              required boolean element;
                            }
                          }
                          optional binary ValueUnsupported (STRING);
                        }
                      }
                    }
                    required int32 DroppedAttributesCount (INTEGER(32,true));
                    required group Events (LIST) {
                      repeated group list {
                        required group element {
                          required int64 TimeSinceStartNano (INTEGER(64,false));
                          required binary Name (STRING);
                          required group Attrs (LIST) {
                            repeated group list {
                              required group element {
                                required binary Key (STRING);
                                required boolean IsArray;
                                required group Value (LIST) {
                                  repeated group list {
                                    required binary element (STRING);
                                  }
                                }
                                required group ValueInt (LIST) {
                                  repeated group list {
                                    required int64 element (INTEGER(64,true));
                                  }
                                }
                                required group ValueDouble (LIST) {
                                  repeated group list {
                                    required double element;
                                  }
                                }
                                required group ValueBool (LIST) {
                                  repeated group list {
                                    required boolean element;
                                  }
                                }
                                optional binary ValueUnsupported (STRING);
                              }
                            }
                          }
                          required int32 DroppedAttributesCount (INTEGER(32,true));
                        }
                      }
                    }
                    required int32 DroppedEventsCount (INTEGER(32,true));
                    required group Links (LIST) {
                      repeated group list {
                        required group element {
                          required binary TraceID;
                          required binary SpanID;
                          required binary TraceState (STRING);
                          required group Attrs (LIST) {
                            repeated group list {
                              required group element {
                                required binary Key (STRING);
                                required boolean IsArray;
                                required group Value (LIST) {
                                  repeated group list {
                                    required binary element (STRING);
                                  }
                                }
                                required group ValueInt (LIST) {
                                  repeated group list {
                                    required int64 element (INTEGER(64,true));
                                  }
                                }
                                required group ValueDouble (LIST) {
                                  repeated group list {
                                    required double element;
                                  }
                                }
                                required group ValueBool (LIST) {
                                  repeated group list {
                                    required boolean element;
                                  }
                                }
                                optional binary ValueUnsupported (STRING);
                              }
                            }
                          }
                          required int32 DroppedAttributesCount (INTEGER(32,true));
                        }
                      }
                    }
                    required int32 DroppedLinksCount (INTEGER(32,true));
                    optional binary HttpMethod (STRING);
                    optional binary HttpUrl (STRING);
                    optional int64 HttpStatusCode (INTEGER(64,true));
                    required group DedicatedAttributes {
                      optional binary String01 (STRING);
                      optional binary String02 (STRING);
                      optional binary String03 (STRING);
                      optional binary String04 (STRING);
                      optional binary String05 (STRING);
                      optional binary String06 (STRING);
                      optional binary String07 (STRING);
                      optional binary String08 (STRING);
                      optional binary String09 (STRING);
                      optional binary String10 (STRING);
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
```

{{< /collapse >}}

For the authoritative schema, refer to `tempodb/encoding/vparquet4/schema.go` and `tempodb/encoding/vparquet5/schema.go`.

### Summary of vParquet5 differences

- Resource-level dedicated columns (Cluster/Namespace/Pod/Container/K8s\*) and span HTTP columns are removed; vParquet5 relies on dynamically assigned dedicated columns only.
- Dedicated attribute columns expand to include integer spares, array attributes, and optional blob configuration for selected columns.
- Additional fields exist for optimization, including span `ChildCount`, rounded start time buckets, and trace-level `ServiceStats` as a list with explicit service names.

## Trace-level attributes

For speed and ease-of-use, we are projecting several values to columns at the trace-level:

- Trace ID - Don't store on each span.
- Root service/span names/StartTimeUnixNano - These are selected properties of the root span in each trace (if there is one). These are used for displaying results in the Grafana UI. These properties are computed at ingest time and stored once for efficiency, so we don't have to find the root span.
- `DurationNano` - The total trace duration, computed at ingest time. This powers the min/max duration filtering in the current Tempo search and is more efficient than scanning the spans duration column. However, it may go away with TraceQL or we could decide to change it to span-level duration filtering too.
- `ServiceStats` - Per-service span and error counts for each trace.

## "Any"-type Attributes

OTLP attributes have variable data types, which is simpler in formats like protocol-buffers, but doesn't translate directly to Parquet.
Each column must have a concrete type.
There are several possibilities here but we chose to have optional values for each concrete type.
Unsupported or mixed-type values are stored as JSON-encoded AnyValue in `ValueUnsupported`.

Only scalar values are stored in the dedicated attribute columns. Arrays and unsupported types remain in the generic attribute columns.

```yaml
repeated group Attrs {
  required binary Key (STRING);
  required boolean IsArray;
  repeated binary Value (STRING);
  repeated int64 ValueInt (INTEGER(64,true));
  repeated double ValueDouble;
  repeated boolean ValueBool;
  optional binary ValueUnsupported (STRING);
}
```

## Compression and encoding

Parquet has robust support for many compression algorithms and data encodings. We've found excellent combinations of storage size and performance with the following:

1. Snappy Compression - Enable on all columns
1. Dictionary encoding - Enable on all string columns. Most strings are very repetitive so this works well to optimize storage size. However, you can greatly speed up search by inspecting the dictionary first and eliminating pages with no matches.
1. Time and duration UNIX nanos - Delta encoding
1. Rarely used columns such as `DroppedAttributesCount` - These columns are usually all zeroes, RLE works well.

### Bloom filters

Parquet has native support for bloom filters. However, Tempo doesn't use them at this time. Tempo already has sophisticated support for sharding and caching bloom filters.

<!-- vale Grafana.GoogleSpacing = YES -->
<!-- vale Grafana.We = YES -->
<!-- vale Grafana.GooglePassive = YES -->
<!-- vale Grafana.GoogleWill = YES -->
