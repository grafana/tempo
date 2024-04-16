---
title: Tune search performance
menutitle: Tune search performance
description: How to tune Tempo to improve search performance.
weight: 90
---

# Tune search performance

Regardless of whether or not you are using TraceQL or the original search API, Tempo searches all of the blocks
in the specified time range.
Depending on your volume, this may result in slow queries.

This document explains how Tempo’s read path works, what controls you have to tune the read path, details of the core configuration options, and how to tune these configuration options to get the most out of your Tempo cluster.

The general advice is to scale your compactors and queriers. Additional queriers can more effectively run jobs in parallel
while additional compactors more aggressively reduce the length of your blocklist and copies of data (if using `RF=3`).

{{< admonition type="note" >}}
All forms of search (TraceQL and tags based) are only supported on the `vParquet` and forward blocks. [v2 blocks]({{< relref "../configuration/parquet#choose-a-different-block-format" >}})
can only be used for trace by id lookup.

Tempo 2.3 and higher support [Dedicated attribute columns]({{< relref "./dedicated_columns" >}}), another great method to improve search performance.
{{< /admonition>}}

## Before you begin

You should understand the basic Tempo architecture.

For more information, refer to the [Tempo architecture](https://grafana.com/docs/tempo/latest/operations/architecture/#query-frontend) to learn about how Tempo works.

### Glossary

query
: A search query, issued by end user. For example, `{ traceDuration > 1s }` is a TraceQL query.

job
: A shard of a search query, the lowest unit of work. A query is broken down into multiple jobs and processed.

Batch
: A group of jobs is called a batch.

frontend
: Another name for query-frontend

querier
: Responsible for executing (processing) the jobs, and sending the results back to query-frontend.

## Tempo query path

Tempo has a query path made up of a query-frontend, querrier, ingester, metrics-generator, and backend.

You can think about each component in a query path as a generic producer and worker model:

* The query-frontend is a producer and it has multiple workers connected to it. The query-frontend takes a single query and shards it into multiple jobs (units of work).
* The queriers are workers. They enqueue work from a queue, process it, and send the results back to the producer (query-frontend).
* The querier either reads data from backend and processes the query, or it delegates the query to ingesters and metrics generators based on the type of job, time range, and query type.

![Tempo query path architecture](/media/docs/tempo/tempo-query-frontend.svg)

### Lifecycle of a query

A search request comes from the user, and it is split into multiple jobs in the query-frontend.

These jobs are then added to a queue and queriers pick up jobs in batches to process.

A single search query can create hundreds of jobs based on the time range and other variables.
These jobs are batched and sent to queriers.

The querier receives the batch and processes the jobs in the batch, builds results, and sends the results of the whole batch back to the query-frontend.

The query-frontend merges and deduplicates the results, and sends them back to the client.

When querier starts, it opens multiple connections to each query-frontend.

Job batches are sent over these connections, and return the results.
This process is synchronized so a single job batch can block a connection.

The number of connections control the number of batches a querier processes concurrently.
The number of connections is controlled by `max_concuccrent_queries` OR `frontend_worker.parallelism`.

## General guidelines

Tuning the search pipeline can be difficult as it requires balancing a number of different configuration parameters. The below tips
can you get your head around the general problem, but the specifics require experimentation.

- Review the query-frontend logs for lines like the following to get a feeling for how many jobs your queries are creating:

  ```
  level=info ts=2023-07-19T19:38:01.354220385Z caller=searchsharding.go:236 msg="sharded search query request stats and SearchMetrics" ...
  ```

- For a single TraceQL query the maximum number of parallel jobs is constrained by:
  - `query_frontend.search.concurrent_jobs`: This is the maximum number of jobs the frontend dispatches for one TraceQL query.
  - `# queriers * querier.max_concurrent_queries * query_frontend.max_batch_size`: This is the maximum job capacity of your Tempo cluster.
  If a given TraceQL query produces less jobs then these two values it should be executed entirely in parallel on the queriers.

- Increasing `querier.max_concurrent_queries` is a great way to get more out of your queriers. However, if queriers are OOMing or saturating other
  resources then this should be lowered. Lowering `query_frontend.max_batch_size` will also reduce the total work attempted by one querier.

## Querier and query-frontend configuration

Queriers and query-frontends have additional configuration related to search of the backend datastore.

### Querier

```yaml
querier:
  # Control the amount of work each querier will attempt. The total number of
  # jobs a querier will attempt this is this value * query_frontend.max_batch_size
  max_concurrent_queries: 20
```

With serverless technologies:

{{< admonition type="note" >}}
Serverless can be a nice way to reduce cost by using it as spare query capacity.
However, serverless tends to have higher variance then simply allowing the queriers to perform the searches themselves.
{{< /admonition >}}

```yaml
querier:

  search:
    # A list of endpoints to query. Load will be spread evenly across
    # these multiple serverless functions.
    external_endpoints:
    - https://<serverless endpoint>

    # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
    # be set to p99 of search requests to reduce long tail latency.
    external_hedge_requests_at: 8s

    # The maximum number of requests to execute when hedging. Requires hedge_requests_at to be set.
    external_hedge_requests_up_to: 2
```

### Query-frontend

[Query frontend]({{< relref "../configuration#query-frontend" >}}) lists all configuration
options.

These suggestions help deal with scaling issues.

```yaml
server:
  # At larger scales, searching starts to feel more like a batch job.
  # Increase the server timeout intervals.
  http_server_read_timeout: 2m
  http_server_write_timeout: 2m

query_frontend:
  # When increasing concurrent_jobs, also increase the queue size per tenant,
  # or search requests will be cause 429 errors. This is the total number of jobs
  # per tenant allowed in the queue.
  max_outstanding_per_tenant: 2000

  # The number of jobs the query-frontend will batch together when passing jobs to the queriers. This value
  # This value * querier.max_concurrent_queries is your the max number of jobs a given querier will try at once.
  max_batch_size: 3

  search:
    # At larger scales, increase the number of jobs attempted simultaneously,
    # per search query.
    concurrent_jobs: 2000

    # The query-frontend will attempt to divide jobs up by an estimate of job size. The smallest possible
    # job size is a single parquet row group. Increasing this value will create fewer, larger jobs. Decreasing
    # it will create more, smaller jobs.
    target_bytes_per_job: 50_000_000
```

### Serverless environment

Serverless isn't required, but with larger loads, serverless can be used to reduce costs.
Tempo has support for Google Cloud Run and AWS Lambda.
In both cases, you can use the following
settings to configure Tempo to use a serverless environment:

```yaml
querier:
  search:
    # A list of external endpoints that the querier will use to offload backend search requests. They must
    # take and return the same value as /api/search endpoint on the querier. This is intended to be
    # used with serverless technologies for massive parallelization of the search path.
    # The default value of "" disables this feature.
    [external_endpoints: <list of strings> | default = <empty list>]

    # If external_endpoints is set then the querier will primarily act as a proxy for whatever serverless backend
    # you have configured. This setting allows the operator to have the querier prefer itself for a configurable
    # number of subqueries. In the default case of 2 the querier will process up to 2 search requests subqueries before starting
    # to reach out to external_endpoints.
    # Setting this to 0 will disable this feature and the querier will proxy all search subqueries to external_endpoints.
    [prefer_self: <int> | default = 2 ]

    # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
    # be set to p99 of external search requests to reduce long tail latency.
    # (default: 4s)
    [external_hedge_requests_at: <duration>]

    # The maximum number of requests to execute when hedging. Requires hedge_requests_at to be set.
    # (default: 3)
    [external_hedge_requests_up_to: <int>]
```

For cloud-specific details:

- [AWS Lambda]({{< relref "./serverless_aws" >}})
- [Google Cloud Run]({{< relref "./serverless_gcp" >}})

## Settings that are safe to increase without major impact.

Scaling up queriers is a safe way to add more query capacity.
At Grafana, we prefer to scale queries horizontally by adding more replicas.
If you see out of memory (OOM) errors, it might be worth scaling the queriers vertically.

There should always be two replicas of query-frontends in a cluster.
If you need to scale, scale query-frontends vertically by adding more CPU and RAM.
Currently, query-frontends aren’t scaled horizontally, but this might change in the future.

The reason we keep only two replicas is because each query-frontend has its own request queue and it also impacts the amount of jobs sent to each querier.
If you add more query-frontends, you need to tune other configuration parameters to account for the change. We decided to keep query-frontend to two replicas for now.

In a dedicated cluster, you can increase `query_frontend.max_outstanding_per_tenant` because the cluster is dedicated to a single customer.
In a shared cluster, you need to use more caution when increasing `querier.max_outstanding_per_tenant`

### Parameter interactions

The `query_frontend` configuration options control how Tempo shards a query into multiple jobs.
These options also control the size of a single job, number of jobs, and the number of jobs added to the work queue at once (concurrently) for queriers to be picked up.
The `max_batch_size` option controls how many jobs are sent to queriers at once in a single batch.

Querier configuration options control processing of these jobs, queries operate at a job level.
They're workers: they pick a job from the queue and execute the job, and return the results back to the query-frontend.
Querier options control how many jobs it processes at once by opening concurrent connections to query-frontends, and each connection processes a single.

## Guidelines on key configuration parameters

The following sections provide recommendations for key configuration parameters.

### `query_frontend.max_outstanding_per_tenant` parameter

The `query_frontend.max_outstanding_per_tenant` parameter decides the maximum number of outstanding requests per tenant per frontend; requests beyond this error with HTTP `429`.

This configuration controls how much load a single tenant can put on a cluster. A return of `429` indicates users to slow down.

This configuration can be used to maintain quality of service (QoS) for tenants by asking tenants to slow down by sending `429` to the tenant that's overloading the system.

#### Guidelines

* In a dedicated cluster with one big tenant, it’s okay to increase the number for `query_frontend.max_outstanding_per_tenant.`
* In a shared cluster with a lot of small tenants, keep the number small.
* If a single tenant is overwhelming the whole cluster, you should decrease this parameter. It reduces the amount of work the tenant can enqueue at once. It starts returning `429` to the tenant.
* If your tenants are complaining about getting `429`, you might want to increase this parameter and scale out queries to handle the query load.
* If you increase other configurations that increase the query throughput or scale queriers, you might want to reduce this parameter to control the amount of work that can be enqueued.


### `query_frontend.max_batch_size` parameter

The number of jobs to batch together in one HTTP request to the querier.
Large search queries over a longer time range produce thousands of jobs.
Instead of sending jobs one by one, batch the jobs and send them at once.
The querier picks up and process them.

Batching pushes jobs faster to the queriers, and reduce the time spent waiting around for jobs to be picked up by the querier.

#### Guidelines

* Default value of `max_batch_size` is set to `5`. In testing, `5` is a good default across all sizes of clusters.
* We DO NOT recommend changing the batch size from the default value of `5`. In testing, `5` was the sweet spot.
* If the batch size is big, you push more jobs at once but it takes longer for the querier to process the batch and return the results back.
* Big batch size will increase the latency of the querier requests, and they might start hitting timeouts of 5xx, which will increase the rate of retries.
* Bigger `max_batch_size` results in pushing too many jobs to the queriers. The jobs then have to be canceled if a query is exited early.

###  `query_frontend.search.concurrent_jobs` parameters

The number of concurrent jobs to execute when searching the backend. This controls how much work is produced concurrently.

To put it another way, if a search job is sharded into 5000 jobs, and `concurrent_jobs` is set to 1000, then Tempo only executes 1000 jobs concurrently.
The other 4000 jobs are processed one by one as the first 100 jobs return the responses.

If Tempo manages to answer the search query without executing all 500 jobs, Tempo exits early and cancels the jobs that weren't started.

#### Guidelines

* This params controls the numbers of concurrent jobs to the number of jobs to put in the queue at once for processing a query.
* If this number is low, big queries that create lots of jobs processed slowly.
* If this number is high, the frontend pushes work faster to queries, and the frontend needs to cancel those jobs if the search exits early because the query is fulfilled.
* If you see high load, spikes, or OOMs in the queriers, you can lower this configuration for a fair QoS and stability.
* Having this number to a high value will allow a single tenant to push lots of work at once and overwhelm the querier. So keeping it to a lower number in a shared cluster allows for fair scheduling.
* This parameter controls how fast jobs are pushed into the queue for queries to be picked up by queries
* Adjusting this can impact your batch size so that metric should be watched while making changes to this configuration, and make sure that actual batch size is around the `max_batch_size`.
* We recommend keeping it high in dedicated clusters with a single big tenant.
* We recommend keeping it low in shared clusters because it ensures that a single tenant can't overwhelm the read path.


### `query_frontend.max_retries` parameter

This option controls the number of times to retry a request sent to a querier.
Current configurations only retry if the return is 5xx (_translated from equivalent gRPC error_) from a querier.

#### Guidelines

* This option controls the retries from query-frontend to queriers.
* High `max_retries` amplifies the slow queriers and result in a lot more load then required.
* Low `max_retries` can lead to users having bad experience due failed queriers, because  a job failed and wasn’t retried.
* If there are too many retries in a cluster, it can be a sign that queriers are struggling to finish work and needs to be scaled up.
* Big jobs (high `target_bytes_per_job` or `max_batch_size`) can result in increased retries. Make the jobs smaller by reducing the `target_bytes_per_job` or `max_batch_size` to reduce the retries. If not, scale up the queriers.
* Don't set `query_frontend.max_retries` to a high number. The default of `2` is a good default.
* Higher retries impact the backend when you have a big query that’s generating lots of jobs.
* High `max_retries` can overwhelm the queriers when they're under heavy load and failing to process the jobs. Retries can snowball, and degrade the query performance.


### `query_frontend.search.target_bytes_per_job` parameter

The target number of bytes for each job to handle when performing a backend search.

This parameter controls the size of a single job, and size of a job dictates how much data a single job reads.
You can tune this to make big or small jobs.

This option controls the upper limit on the size of a job, and can be used as a proxy for  how much data a single search job scans or the amount of work a querier needs to do to process a job.

#### Guidelines

* Setting this to a small value produces too many jobs, and results in more overhead, setting it too high produces big jobs. Queriers might struggle to finish those jobs and it can lead to high latency.
* In testing, 100MB to 200MB is a sweet spot for this configuration, and works best across different sizes of clusters.
* We recommend keeping this fixed within the recommended range and not changing it.


### `querier.search.prefer_self` parameter

{{< admonition type="note" >}}
This configuration only applies to `tempo-serverless`.
{{< /admonition >}}

This setting controls the number of job the querier will process before spilling over the `search_external_endpoints` (tempo-serverless).

#### Guidelines

* In testing at Grafana, serverless suffered from cold starts problems. If your query load is predictable, serverless isn't recommended.
* Increase the value of `prefer_self` if you want to process more jobs in the querier and spill out in extreme cases.
* Setting this to a very big number is as good as turning it off because the querier tries to process all the jobs and it never spills over to serverless.
* If we set this to a low value, we spill more jobs to serverless, even when queriers have capacity to process the job, and due to cold start, query latency increases.

### `querier.frontend_worker.parallelism`

Number of simultaneous queries to process per query-frontend or query-scheduler. This configuration controls the number of concurrent requests per query-frontend a querier process.

If parallelism is set to 5 and there are two query-frontends running, then a querier process opens 5 connections to each query-frontend, and in total of 10 connections.

One batch (or job, if batching is disabled) is processed per connection, so this controls the maximum number of concurrent jobs processed.

A single batch is processed in sync and size of the batch is controlled by `max_batch_size`, a connection is blocked until it's process and returns the result of a batch.

You can also disable it and use `max_concurrent_connections` but we use this to make sure we queriers are picking up work from both query-frontends and jobs are being scheduled fairly.

If you want to make sure you always have connections defined in this config, you should set  `match_max_concurrent: false` in your configuration to ensure you are not limited by `max_concurrent_connections`.

#### Guidelines

* Maximum number of jobs a querier processes when using parallelism is equal to `parallelism * query_frontend replicas * max_batch_size`.
* As you add more queriers, connections to an individual frontend increase. Query-frontend has a shared lock on these connections so if you see issues when you scale out queriers, lower the parallelism to reduce contention.
* It’s recommended to set `match_max_concurrent: false` and not set `max_concurrent_queries` when using parallelism.

### querier.max_concurrent_queries

This controls the maximum number of jobs a querier processes concurrently. It doesn't distinguish between the types of queries. These jobs can be from two different queries, or different tenants as well.

Querier processes these in sync, blocks the connection, and once it returns the results, it picks up a new job from the query-frontend queue.

#### Guidelines

* The `max_concurrent_queries` parameter controls the number of queries that are processed by querier at once (concurrently).
* if queriers are under-utilized, increase this value to process more jobs concurrently and process the queries faster.
* If queriers are over utilized, seeing spikes, or OOMs, you can reduce this config to do less work per queriers and scale out queriers to add more capacity to the cluster.
* This setting depends on the resources, in small queriers, it should be set to a lower number.
* If you have large queriers, this configuration should be set to a higher value to fully utilize the capacity of the querier.

## Querier memory sizing

If you are seeing that your queries are using more memory then you prefer, reduce the amount of work a querier is doing concurrently. That reduces resource usage.

If you want your queries to use more memory then they're currently using, increase the amount of work a querier is doing concurrently and it increases the resource usage.

You can tune `query_frontend.search.target_bytes_per_job`, `querier.frontend_worker.parallelism`, and `querier.max_concurrent_queries` to tune the amount of work a querier is doing concurrently.

Queriers memory request roughly translates to job size times the concurrent work and some buffer.

## Scaling cache

Tempo can be configured to use multiple caches for different types of data.
When configured, Tempo uses these caches to improve the performance of queries.

Here are few general heuristics to know when to scale a cache:

* Look at cache latency. If cache latency is hitting `cache_timeout`, that means that cache is under scaled and it’s taking too long to read or write, scale the cache.
* Scale the cache if a cache has a high eviction rate. The cache might be under provisioned.
* Lower level cache like bloom cache, parquet-page cache, parquet-footer cache sees higher hit rates (usually around 90% of above). If you have a consistent query traffic and these lower level caches have low hit rate, they're undervalued and needs to be scaled up.
* Higher level cache like frontend-search cache has a low hit rate and is only useful when the same query is being repeated. Size these according to the amount of data you want you want to cache
* Cache sizes are also dictated by how much data you want to cache at each tier, it’s better to cache more at lower level caches because they have higher hit rate and is useful across queries.