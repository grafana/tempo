---
title: Tune search performance
menutitle: Tune search performance
description: How to tune Tempo to improve search performance.
weight: 90
---

# Tune search performance

Regardless of whether or not you are using TraceQL or the original search API, Tempo will search all of the blocks
in the specified time range.
Depending on your volume, this may result in slow queries.
This document contains suggestions for tuning your backend to improve performance.

General advice is to scale your compactors and queriers. Additional queriers can more effectively run jobs in parallel
while additional compactors will more aggressively reduce the length of your blocklist and copies of data (if using RF=3).

>**Note:** All forms of search (TraceQL and tags based) are only supported on the `vParquet` and forward blocks. [v2 blocks]({{< relref "../configuration/parquet#choose-a-different-block-format" >}})
can only be used for trace by id lookup.

>**Note:** Tempo 2.3 and higher supports [Dedicated attribute columns]({{< relref "./dedicated_columns" >}}) which is another great method to improve search performance.

## General guidelines

Tuning the search pipeline can be difficult as it requires balancing a number of different configuration parameters. The below tips
can you get your head around the general problem, but the specifics require experimentation.

- Review the query-frontend logs for lines like the following to get a feeling for how many jobs your queries are creating:
  ```
  level=info ts=2023-07-19T19:38:01.354220385Z caller=searchsharding.go:236 msg="sharded search query request stats and SearchMetrics" ...
  ```

- For a single TraceQL query the maximum number of parallel jobs is constrained by:
  - `query_frontend.search.concurrent_jobs`: This is the maximum number of jobs the frontend will dispatch for one TraceQL query.
  - `# queriers * querier.max_concurrent_queries * query_frontend.max_batch_size`: This is the maximum job capacity of your Tempo cluster.
  If a given TraceQL query produces less jobs then these two values it should be executed entirely in parallel on the queriers.

- Increasing `querier.max_concurrent_queries` is a great way to get more out of your queriers. However, if queriers are OOMing or saturating other
  resources then this should be lowered. Lowering `query_frontend.max_batch_size` will also reduce the total work attempted by one querier.

## Configuration

Queriers and query frontends have additional configuration related
to search of the backend datastore.

### Querier

```
querier:
  # Control the amount of work each querier will attempt. The total number of
  # jobs a querier will attempt this is this value * query_frontend.max_batch_size
  max_concurrent_queries: 20
```

With serverless technologies:

>**Note:** Serverless can be a nice way to reduce cost by using it as spare query capacity. However, serverless tends to have higher variance then simply allowing the queriers to perform the searches themselves.

```
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


### Query frontend

[Query frontend]({{< relref "../configuration#query-frontend" >}}) lists all configuration
options.

These suggestions will help deal with scaling issues.

```
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

    # The query frontend will attempt to divide jobs up by an estimate of job size. The smallest possible
    # job size is a single parquet row group. Increasing this value will create fewer, larger jobs. Decreasing
    # it will create more, smaller jobs.
    target_bytes_per_job: 50_000_000
```

## Serverless environment

Serverless is not required, but with larger loads, serverless can be used to reduce costs. 
Tempo has support for Google Cloud Run and AWS Lambda. In both cases, you will use the following
settings to configure Tempo to use a serverless environment:

```
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

See here for cloud-specific details:

- [AWS Lambda]({{< relref "./serverless_aws" >}})
- [Google Cloud Run]({{< relref "./serverless_gcp" >}})

## Caching

If you have set up an external cache (redis or memcached) in in your storage block you can also use it to cache
parquet footers using the following configuration:

```
storage:
  trace:
    search:
      cache_control:
        footer: true
```
