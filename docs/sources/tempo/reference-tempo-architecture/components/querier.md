---
title: Querier
menuTitle: Querier
description: How the querier executes queries against live-stores and object storage.
weight: 600
topicType: concept
versionDate: 2026-03-20
---

# Querier

The querier is the worker component that executes query jobs dispatched by the query frontend. It fetches trace data from both live-stores (for recent data) and object storage (for historical data), then returns results to the query frontend for merging.

## Why the querier exists

Trace data in Tempo lives in two places: recent data in live-stores and historical data in object storage blocks.
The querier bridges both sources, fetching and merging data so that the query frontend doesn't need to know where data lives.
This separation lets you scale query execution independently from query planning and result merging.

## Query execution

When a querier receives a batch of jobs from the query frontend, it processes each job by determining where the relevant data lives.

For recent data, the querier contacts live-stores that own the partitions covering the query's time range. Live-stores respond with any matching spans held in memory or their local WAL.

For historical data, the querier reads block metadata from the blocklist, identifies which blocks may contain matching data, and fetches the relevant portions from object storage. Bloom filters efficiently skip blocks that don't contain the requested trace IDs.

Results from both sources are combined and returned to the query frontend.

## Live-store queries

The querier uses the partition ring to determine which live-stores to contact for a given query. For zone-aware deployments, the querier only needs a response from one live-store per partition (read quorum of 1).

If a live-store is unavailable, the querier falls back to the live-store in the other availability zone. If no live-store is available for a partition, recent data for that partition is temporarily unavailable, but historical queries still work.

## Backend queries

For historical data, the querier consults the blocklist (maintained by backend workers) to find blocks in the relevant time range. It uses bloom filters to quickly eliminate blocks that don't contain the target trace ID, fetches matching block data from object storage (using caching where configured), reads the Parquet data, and applies any TraceQL filters.

### Caching

Queriers benefit significantly from caching. Tempo supports multiple cache tiers.

The frontend search cache caches query results at the frontend level. It has a low hit rate and is mainly useful for repeated queries. The Parquet page cache caches individual Parquet pages with a high hit rate, useful across many different queries. The bloom filter cache caches bloom filters used for trace ID lookups, also with a high hit rate.

Lower-level caches (bloom, Parquet page) have higher hit rates and should be sized more generously than higher-level caches.

## Concurrency

The number of jobs a querier processes concurrently is controlled by `max_concurrent_queries` (the maximum number of jobs processed at once) or `frontend_worker.parallelism` (the number of connections to each query frontend, which determines concurrent batch processing).

Increasing concurrency makes queriers process more jobs in parallel but increases memory usage. If queriers run out of memory, reduce concurrency and scale horizontally instead.

### Memory sizing

Querier memory usage roughly scales with: `job_size * querier_concurrency + buffer`. You can tune this by adjusting `target_bytes_per_job` (at the frontend), `max_concurrent_queries` (at the querier), and `frontend_worker.parallelism` (which affects how many batches the querier processes at once).

## Related resources

Refer to the [querier configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#querier) for the full list of options.
