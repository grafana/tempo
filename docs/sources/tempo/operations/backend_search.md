---
title: Backend search
weight: 90
---

# Backend search

Regardless of whether or not you are using TraceQL or the original search api Tempo will search all of the blocks
in the specified time range. Depending on your volume this may result in quite slow queries. This document contains
some suggestions for tuning your backend to improve performance.

General advice is to scale your compactors and queriers. Additional queriers can more effectively run jobs in parallel
while additional compactors will more aggressively reduce the length of your blocklist and copies of data (if using RF=3).

>**Note:** All forms of search (TraceQL and tags based) are only supported on the `vParquet` and forward blocks. [v2 blocks]({{< relref "../configuration/parquet#disable-parquet" >}})
can only be used for trace by id lookup.

## Configuration

Queriers and query frontends have additional configuration related
to search of the backend datastore.

### Querier

Without serverless technologies:

```
querier:
  # Greatly increase the amount of work each querier will attempt
  max_concurrent_queries: 20
```

With serverless technologies:

```
querier:
  # The querier is only a proxy to the serverless endpoint.
  # Increase this greatly to permit needed throughput.
  max_concurrent_queries: 100

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
  # or search requests will be cause 429 errors.
  max_outstanding_per_tenant: 2000

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

Serverless is not required, but with larger loads, serverless is recommended to reduce costs and
improve performance. If you find that you are scaling up your quantity of queriers, yet are not
achieving the latencies you would like, switch to serverless.

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

- [AWS Lambda]({{< relref "serverless_aws/" >}})
- [Google Cloud Run]({{< relref "serverless_gcp/" >}})

## Caching

If you have set up an external cache (redis or memcached) in in your storage block you can also use it to cache
parquet footers using the following configuration:

```
storage:
  cache_control:
    footer: true
```