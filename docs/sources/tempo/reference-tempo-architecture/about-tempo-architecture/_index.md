---
title: About Tempo's architecture
menuTitle: About the architecture
description: Understand the design philosophy and data flow in Tempo.
weight: 100
topicType: concept
versionDate: 2026-03-20
---

# About Tempo's architecture

Grafana Tempo is a distributed tracing backend designed for high-volume trace ingestion and querying at scale.
Tempo 3.0 introduces a fundamentally new architecture that decouples the write and read paths using a Kafka-compatible message queue as a durable intermediary.

## Design philosophy

Tempo's architecture is built around several key principles.

Separate components handle writing trace data to storage and serving queries.
You can scale writes and reads independently, and a failure in one path doesn't affect the other.

Kafka serves as a durable write-ahead log (WAL) between distributors and downstream consumers.
Once Kafka acknowledges a write, the data is safe.
This replaces the previous in-process ingestion WAL that lived on local disks,
while live-stores still use a separate local WAL for quickly available search.

Because Kafka provides durability on the write path,
Tempo doesn't need to replicate data across multiple instances.
This replication factor of 1 significantly reduces cost and query complexity.

Tempo uses Apache Parquet as its data format,
storing trace data in a columnar layout that enables efficient querying of specific attributes without reading entire traces.

## Write path

The write path gets trace data from instrumented applications into long-term object storage.

```
Application -> Distributor -> Kafka -> Block-builder -> Object storage
```

1. Distributors receive trace data over OTLP (or other supported protocols),
validate it against rate limits, shard traces by trace ID, and write records to Kafka partitions.
2. Kafka durably stores the records.
The write is acknowledged to the client as soon as Kafka confirms receipt.
3. Block-builders consume records from Kafka, organize spans into blocks in Apache Parquet format, 
and flush those blocks to object storage.

The block-builder operates on a consumption cycle: it reads a batch of records from Kafka,
builds blocks from them, flushes the blocks to object storage, and commits the offset back to Kafka.
Each cycle produces a clean cut of data.
Traces that span multiple cycles have their spans split across blocks, which the query path handles at query time.

## Read path

The read path serves queries by combining recent data from live-stores with historical data from object storage.

```
Client -> Query frontend -> Querier -> Live-store (recent data)
                                    -> Object storage (historical data)
```

1. The query frontend receives a query, shards it into parallel jobs, and distributes them to queriers.
2. Queriers execute jobs by fetching data from two sources:
live-stores for recent data (typically the last 30 minutes to 1 hour) and object storage for historical data,
using bloom filters and indexes for efficient block lookups.
3. The query frontend merges results from all queriers and returns the response.

## Live-stores and the recent data window

Live-stores are the read-path component responsible for serving recent trace data.
They consume from Kafka independently of block-builders and write traces to temporary on-disk blocks,
making them available for queries seconds after they're consumed.

Live-stores exist because there's a gap between when you write data to Kafka
and when the block-builder flushes it to object storage.
During this window, the only way to query that data is through the live-store.

Live-stores own the partition lifecycle within Tempo.
They manage a partition ring that tracks which partitions are active and which live-stores own them.
This is separate from Kafka's internal partition management.
Refer to the [partition ring](../partition-ring/) documentation for details.

## How the paths connect

The write and read paths connect through two mechanisms:

1. Kafka is the shared source of truth. Both block-builders and live-stores consume from the same Kafka partitions,
but they track their own consumer offsets independently.
2. Object storage is where the paths converge. Block-builders write blocks there; queriers read from there.

Even if a block-builder is down or slow, live-stores continue serving recent data.
If a live-store restarts, it replays from Kafka to rebuild its in-memory state.
The two paths are resilient to independent failures.

## Component summary

| Component | Path | Responsibility |
|---|---|---|
| Distributor | Write | Receives traces, validates limits, writes to Kafka |
| Kafka | Write | Durable message queue between distributor and consumers |
| Block-builder | Write | Consumes from Kafka, builds Parquet blocks, flushes to object storage |
| Live-store | Read | Consumes from Kafka, serves recent data to queriers |
| Query frontend | Read | Shards queries into jobs, distributes to queriers, merges results |
| Querier | Read | Executes query jobs against live-stores and object storage |
| Backend scheduler/worker | Maintenance | Compacts and deduplicates blocks, enforces retention |
| Metrics-generator | Optional | Consumes from Kafka, derives metrics from traces |
