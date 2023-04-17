---
title: Inspect Apache Parquet data
menuTitle: Inspect Parquet data
weight: 75
---

# Inspect Apache Parquet data

Apache Parquet is a column-oriented format and is the default block format for Tempo 2.0.
Refer to the [Parquet configuration options]({{< relref "../../configuration/parquet.md" >}}) for more information.

You can use `parquet-tools` to query and inspect Parquet files. There are multiple distributions of parquet-tools available. Most of the examples on this page should work with different distributions.

## Examples

Examples of working with Tempo parquet blocks and popular parquet tooling.


This command shows the schema of a Tempo parquet block using the parquet-tools command:

```bash
$ parquet-tools schema data.parquet
message Trace {
  required binary TraceID;
  repeated group rs {
    required group Resource {
      repeated group Attrs {
        required binary Key (STRING);
        optional binary Value (STRING);
        .
        .
        .
```

This command dumps all column sizes for a Tempo block using the `parquet-tools` command:

```bash
$ parquet-tools column-size data.parquet | sort
DurationNanos-> Size In Bytes: 799849 Size In Ratio: 0.0011075386
EndTimeUnixNano-> Size In Bytes: 952660 Size In Ratio: 0.0013191337
RootServiceName-> Size In Bytes: 108145 Size In Ratio: 1.4974672E-4
RootSpanName-> Size In Bytes: 212444 Size In Ratio: 2.9416793E-4
StartTimeUnixNano-> Size In Bytes: 953946 Size In Ratio: 0.0013209144
TraceID-> Size In Bytes: 3076838 Size In Ratio: 0.00426045
TraceIDText-> Size In Bytes: 2979464 Size In Ratio: 0.004125618
rs.Resource.Attrs.Key-> Size In Bytes: 5404392 Size In Ratio: 0.0074833785
rs.Resource.Attrs.Value-> Size In Bytes: 20337501 Size In Ratio: 0.028161023
rs.Resource.Attrs.ValueArray-> Size In Bytes: 1838177 Size In Ratio: 0.0025452955
.
.
.
```

Dump all service names containing "tempo" from the block:

```bash
$ parquet-tools dump -c rs.Resource.ServiceName data.parquet | grep tempo
value 60:     R:0 D:1 V:tempo-compactor
value 6343:   R:0 D:1 V:tempo-ingester
value 10773:  R:0 D:1 V:tempo-compactor
value 14397:  R:0 D:1 V:tempo-compactor
value 16824:  R:0 D:1 V:tempo-ingester
value 17842:  R:0 D:1 V:tempo-query-frontend
value 22749:  R:0 D:1 V:tempo-gateway
value 25311:  R:0 D:1 V:tempo-compactor
value 30070:  R:0 D:1 V:tempo-compactor
value 30451:  R:0 D:1 V:tempo-query-frontend
value 32755:  R:1 D:1 V:tempo-gateway
value 32866:  R:0 D:1 V:tempo-compactor
value 34998:  R:0 D:1 V:tempo-querier
value 37242:  R:0 D:1 V:tempo-querier
value 40643:  R:0 D:1 V:tempo-compactor
value 42229:  R:0 D:1 V:tempo-compactor
value 42241:  R:0 D:1 V:tempo-gateway
value 43234:  R:0 D:1 V:tempo-gateway
value 43244:  R:0 D:1 V:tempo-compactor
```