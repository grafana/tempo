---
title: Backend search
weight: 9
---

# Backend search

<span style="background-color:#f3f973;">Search is an experimental feature.</span>

Backend search is not yet mature. It can therefore be operationally more complex.
The defaults do not yet support Tempo well.

Search of the backend datastore will likely exhibit poor performance
unless you make some the of the changes detailed here.

## Configuration

Queriers and query frontends have additional configuration related
to search of the backend datastore. 
Some defaults are currently tuned for a search by trace ID.

### All components

```
# Enable search functionality
search_enabled: true
```

### Querier

Without serverless technologies:

```
querier:
  # Greatly increase the amount of work each querier will attempt
  max_concurrent_queries: 20
```

With serverless technologies:

```
search_enabled: true
querier:
  # The querier is only a proxy to the serverless endpoint.
  # Increase this greatly to permit needed throughput.
  max_concurrent_queries: 100

  search:
    # A list of endpoints to query. Load will be spread evenly across
    # these multiple serverless functions.
    external_endpoints:
    - https://<serverless endpoint>
```

### Query frontend

[Query frontend](../../configuration#query-frontend) lists all configuration
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

    # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
    # be set to p99 of search requests to reduce long tail latency.
    hedge_requests_at: 5s

    # The maximum number of requests to execute when hedging. Requires hedge_requests_at to be set.
    hedge_requests_up_to: 3
```

## Serverless environment

Serverless is not required, but with larger loads, serverless is recommended to reduce costs and
improve performance. If you find that you are scaling up your quantity of queriers, yet are not 
acheiving the latencies you would like, switch to serverless.

Tempo has support for Google Cloud Functions and AWS Lambda. In both cases you will use the following
settings to configure Tempo to use the serverless:

```
querier:
  search:
    # A list of external endpoints that the querier will use to offload backend search requests. They must  
    # take and return the same value as /api/search endpoint on the querier. This is intended to be
    # used with serverless technologies for massive parrallelization of the search path.
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

- [AWS Lambda](./serverless_aws.md)
- [Google Cloud Functions](./serverless_gcp.md)
