---
Authors: Martin Disibio (@mdisibio), Annanay Agarwal (@annanay25)
Created: April 2022
Last updated: 2022-04-22
---

# Parquet

## Summary

This design document describes adding a new columnar block format to Tempo based on Apache Parquet.  A columnar block format has many advantages within Tempo such as enabling faster search but also downstream by enabling a large ecosystem of tools access to the underlying trace data.

## Context
Over the last few months, the Tempo Team has invested a lot of effort in implementing efficient search queries over trace data spanning long time ranges. We mainly worked on two major initiatives to enable search and summarise our findings below:

1. **Scaling up the existing Protobuf-based blocks**.  Tempo currently leverages massively-parallel serverless functions to perform full backend-search.  However even with thousands of functions, Tempo search is not able to achieve the search speeds that we would like on larger datasets.  Each trace is stored as an individual proto message and must be unmarshaled to look for things like service names and tags.  Even with scanning speeds of 60 gb/s from object storage, the large I/O makes it infeasible to query >24h periods over larger datasets.

2. **Implementing a Flatbuffer-based storage format**.  Tempo currently uses flatbuffer-based data to power ingester search.  Flatbuffers improve search speeds by incurring no deserialization cost as the on-disk format matches the in-memory format.  However, the current ingester data is very limited to the current application and stores only a subset of searchable data.  Storing a 100% roundtrippable version of traces proved to be ~50% larger than the existing block size and is not feasible. This is due to the design of flatbuffers for alignment of various values, padding, and inability to use space-efficient data encodings such as variable-width integers.

## Design Goals and Requirements
The main design goals in choosing a new block format are to increase the speed and efficiency of Tempo search, and power the upcoming TraceQL language for querying and extracting metrics from traces.

We also had the following requirements of the new format:
* Roundtrippable with OTLP - A new block format must support full trace read/write, so it must be able to be converted from OTLP and back again.
* More efficient search - Reduce i/o
* Faster search - There's no point in having a new format that is slower
* Similar or better block size - No significant increase in storage requirements for the same data
* Object storage - Works with object storage as the only dependency
* Shardable - Ability to scan a block by multiple queriers concurrently, with no upper limit (i.e. 1000's of workers)

### Why parquet
Parquet fits all of the requirements:
* Roundtrippable - Support for hiearchical data means regenerating OTLP is straightformat (more on the nested schema below)
* More efficient - only read the columns needed for the search
* Faster - only read the columns needed
* Similar or better block size - Current prototype has ~5% smaller block size for parquet
* Object storage - Parquet is a file-based approach which translates well to Tempo's block-based design
* Shardable - Parquet naturally includes sub-block structures such as row groups and column chunks which can be processed independently.

## Schema

### Fully Nested vs. Span-oriented
There are two overall approaches to a columnar schema: fully nested or span-oriented.  Span-oriented means a flattened schema where traces are destructured into rows of spans.  A fully nested schema means the current trace structures such as Resource/InstrumentationLibrary/Spans/Events are preserved (nested data is natively supported in Parquet).  In both cases individual leaf values such as span name and duration are individual columns.

We chose the nested schema for several reasons. (a) The block size is much smaller for the nested schema. This is due to the high data duplication incurred when flattening resource-level attributes such as service.name to each individual span. (b) A flat schema is not truly "flat" because each span still contains nested data such as attributes and events.  (c) Nested schema is much faster to search for resource-level attributes because the resource-level columns are very small (1 row for each batch) (d) Translation to/from OTLP is very straightforward (e) Easily add computed columns (ex: Trace duration) at multiple levels such as per-trace, per-batch, etc.

### Static vs Dynamic columns
Additionally there is another layer to the schema which is dynamic vs static columns.  A dynamic schema means storing each attribute such as "service.name" and "http.status_code" as its own column, and the columns in each parquet file can be different.  A static schema means it is irresponsive to the shape of the data, and all attributes are stored in generic key/value containers.

The dynamic schema is the ultimate "dream" for a columnar format but it is too complex for a first release. However the benefits of that approach are also too good to pass up, so we propose a hybrid approach.  It is primarily a static schema but with some dynamic columns extracted from trace data based on some heuristics of frequently queried attributes.  We plan to continue investing in this direction to implement a fully dynamic schema where trace attributes are blown out into independent parquet columns at runtime.

### Proposed Schema
Here is the proposed parquet schema. It is mainly a directly transation of OTLP but with some key differences. We will discuss details and rationale of several areas below:

```
message Trace {
    # Trace-level attributes
    required binary TraceID (STRING);
    required binary RootServiceName (STRING);
    required binary RootSpanName (STRING);
    required int64 StartTimeUnixNano (INT(64,false));
    required int64 DurationNanos (INT(64,false));

    repeated group ResourceSpans {
        required group Resource {
            repeated group Attrs {
                required binary Key (STRING);
                optional binary Value (STRING);
                optional binary ValueArray;
                optional boolean ValueBool;
                optional double ValueDouble;
                optional int64 ValueInt (INT(64,true));
                optional binary ValueKVList;
            }

            # Dedicated columns for common attributes
            required binary ServiceName (STRING);
            optional binary Cluster (STRING);
            optional binary Container (STRING);
            optional binary Namespace (STRING);
            optional binary Pod (STRING);
            optional binary K8sClusterName (STRING);
            optional binary K8sContainerName (STRING);
            optional binary K8sNamespaceName (STRING);
            optional binary K8sPodName (STRING);
        }
        repeated group InstrumentationLibrarySpans {
            repeated group Spans {
                required binary ID;
                required int32 DroppedAttributesCount (INT(32,true));
                required int32 DroppedEventsCount (INT(32,true));

                required int32 Kind (INT(8,true));
                required binary Name (STRING);
                required binary ParentSpanID (STRING);
                required int64 StartUnixNanos (INT(64,false));
                required int64 EndUnixNanos (INT(64,false));
                required binary TraceState (STRING);
                required int32 StatusCode (INT(8,true));
                required binary StatusMessage (STRING);

                repeated group Attrs {
                    required binary Key (STRING);
                    optional binary Value (STRING);
                    optional binary ValueArray;
                    optional boolean ValueBool;
                    optional double ValueDouble;
                    optional int64 ValueInt (INT(64,true));
                    optional binary ValueKVList;
                }

                repeated group Events {
                    repeated group Attrs {
                        required binary Key (STRING);
                        required binary Value (STRING);
                    }
                    required int32 DroppedAttributesCount (INT(32,true));
                    required binary Name (STRING);
                    required int64 TimeUnixNano (INT(64,false));
                }

                # Dedicated columns for common span attributes
                optional binary HttpMethod (STRING);
                optional binary HttpUrl (STRING);
                optional int64 HttpStatusCode (INT(64,true));
            }

            required group InstrumentationLibrary {
                required binary Name (STRING);
                required binary Version (STRING);
            }
        }
    }
}
```

### Trace-level attributes
For speed and ease-of-use, we are projecting several values to columns at the trace-level:

* Trace ID - Don't store on each span.  It is a UTF-8 hexadecimal string for user-friendliness. We could save a few bytes by storing the underlying 16 bytes, but this makes it easier to work with for downstream applications.
* Root service/span names/StartTimeUnixNano - These are selected properties of the root span in each trace (if there is one). These are used for display of results in the Grafana UI. Computed at ingest time and stored once for efficiency, so we don't have to find the root span.
* DurationNanos - The total trace duration, computed at ingest time. This powers the min/max duration filtering in the current Tempo search and is more efficient than scanning the spans duration column. However, it may go away with TraceQL or we could decide to change it to span-level duration filtering too.


### Dedicated columns
Projecting attributes to their own columns has massive benefits for speed and size, and these are too good to pass up on.  Therefore we are taking an opinionated approach and projecting some common attributes to their own columns.  All other attributes are stored in the generic key/value maps and are still searchable, but not as quickly.  We chose these attributes based on what we commonly use ourselves (scratching our own itch), but we think they will be useful to most workloads.  There is an associated cost for unused attributes as it will still store a column of nulls, however the cost should be minimal.  We can be more generous with unused resource-level attributes because their overhead is the smallest of all, we are finding it to be ~.05% (0.0005) of the total block size.

Resource-level:
* `service.name`
* `cluster` and `k8s.cluster.name`
* `namespace` and `k8s.namespace.name`
* `pod` and `k8s.pod.name`
* `container` and `k8s.container.name`

Span-level
* `http.method`
* `http.url`
* `http.status_code` (int)


### "Any"-type Attributes
OTLP attributes have variable data types, which is easy to accomplish in formats like protocol-buffers, but does not translate directly to Parquet.  Each column must have a concrete type.

There are several possibilities here but we chose to have an optional values for each concrete type.  We aren't 100% sold on this approach and open to feedback.  Array and KeyValueList types are stored as JSON-encoded strings. The data is portable and still searchable at extremely basic levels.

```
repeated group Attrs {
    required binary  Key (STRING);

    # Only one of these will be set
    optional binary  Value (STRING);
    optional boolean ValueBool;
    optional double  ValueDouble;
    optional int64   ValueInt (INT(64,true));
    optional binary  ValueArray (STRING);
    optional binary  ValueKVList (STRING);
}
```

Pros:

* Preserves the functionality of parquet column min/max statistics which can be leveraged during search
* Native data types should be more efficient to store and process

Cons:

* Many nulls - as each attribute only populates 1 column, the other 5 are guaranteed to be null. This has a non-trivial impact on storage size.

### Event Attributes
Span event attributes are stored as JSON-encoded strings in a generic key/value map. This is by far the most space-efficient encoding and the trade-off of decreased searchability seems worthwhile.  Storing event attributes this way reduces the block size ~16% for our dataset, which is huge.  There are currently no use cases to search event attributes, but we can revisit this in the future if needed.

```
repeated group Attrs {
    required binary Key (STRING);
    required binary Value (STRING);
}
```

### Compression and Encoding
Parquet has robust support for many compression algorithms and data encodings. We have found excellent combinations of storage size and performance with the following:

1. LZ4 Compression - Enable on all columns
2. Dictionary encoding - Enable on all string columns (including byte array ParentSpanID). Most strings are very repetitive so this works well to optimize storage size.  However we can greatly speed up search by inspecting the dictionary first and eliminating pages with no matches.
3. Time and duration unix nanos - Delta encoding
4. Rarely used columns such as DroppedAttributesCount - These columns are usually all zeroes, RLE works well.

### Bloom Filters
Parquet has native support for bloom filters however we are not using them at this time.  Tempo already has sophisticated support for sharding and caching bloom filters, and we will continue to leverage that for now.

## Results from Local Testing
Here are some interesting column sizes from a block of size ~600MB containing ~150K traces:

```
column.Trace.DurationNanos                                                    154414 values  0.64 MB
column.Trace.ResourceSpans.Resource.ServiceName                              2672245 values  2.01 MB
column.Trace.ResourceSpans.InstrumentationLibrarySpans.Spans.HttpStatusCode 10428415 values  6.35 MB 
column.Trace.ResourceSpans.InstrumentationLibrarySpans.Spans.ID             10428415 values 82.37 MB  # this is never used!
```

Some super early benchmarks on local SSDs:

* Searching protobuf data for a simple query `cluster=ops and minDuration=1s`

```
=== RUN   TestSearchProto
Traces : 55
Traces inspected: 154414
--- PASS: TestSearchProto (21.11s)
```

* Same query on Parquet blocks

```
=== RUN   TestSearchPipelineStatic
Traces : 55
Traces inspected: 154414
Reads  : 290 6.65 MB
--- PASS: TestSearchPipelineStatic (0.18s)
```

## Implementation
We have been refactoring Tempo recently in anticipation of this, therefore Parquet-formatted blocks should be as easy as creating a new folder `/tempodb/encoding/vparquet` and implementing the `VersionedEncoding` interface. Tempo's parquet support will be based on the new library https://github.com/segmentio/parquet-go which we have used for all prototyping and have had excellent success with.

### Write path

* Creating blocks: The ingester will write and flush parquet-formatted blocks. The compactor will only compact like-encoded-blocks, i.e. parquet to parquet. This seems better than having the ingester continue to flush proto and the compactor to convert them.

*  One tunable is the row group size.  Our index pages target something like 256K, but row groups are typically much larger, 50-100MB.  Not sure about the best value here, will need to experiment.

### Read path

* Trace Search: We will implement a query engine to search over the columns involved in the search and join results in memory to output valid traces matching all query parameters.

* Trace Lookup: Our current index pages will be replaced by scanning over the `TraceID` column in the Parquet block. This will provide us with the corresponding row number that matches the queried ID.

* Concurrent search: We will shard searches based on RowGroups and they will work similarly to how we shard on index pages. The metadata for each block will know how many there are and the query-frontend can used them to divvy up tasks, and then the queriers can jump directly to them.
