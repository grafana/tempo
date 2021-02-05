# Runbook

This document should help with remediating operational issues in Tempo.

## TempoRequestErrors
## TempoRequestLatency

Aside from obvious errors in the logs the only real lever you can pull here is scaling.  Use the Reads or Writes dashboard 
to identify the component that is struggling and scale it up.  It should be noted that right now quickly scaling the 
Ingester component can cause 404s on traces until they are flushed to the backend.  For safety you may only want to 
scale one per hour.  However, if Ingesters are falling over, it's better to scale fast, ingest successfully and throw 404s 
on query than to have an unstable ingest path.  Make the call!

The Query path is instrumented with tracing (!) and this can be used to diagnose issues with higher latency. View the logs of
the Query Frontend, where you can find an info level message for every request. Filter for requests with high latency and view traces.

The Query Frontend allows for scaling the query path by sharding queries. There are a few knobs that can be tuned for optimum
parallelism -
- Number of shards each query is split into, configured via
    ```
    query_frontend:
        query_shards: 10
    ```
- Number of Queriers (each of these process the sharded queries in parallel). This can be changed by modifying the size of the
Querier deployment. More Queriers -> faster processing of shards in parallel -> lower request latency.

- Querier parallelism, which is a combination of a few settings:

    ```
    querier:
      max_concurrent_queries: 10
      frontend_worker:
          match_max_concurrent: true  // true by default
          parallelism: 5              // parallelism per query-frontend. ignored if match_max_concurrent is set to true

    storage:
      trace:
        pool:
          max_workers: 100
    ```

MaxConcurrentQueries defines the total number of shards each Querier processes at a given time. By default, this number will
be split between the query frontends, so if there are N query frontends, the Querier will process (Max Concurrent Queries/ N)
queries per query frontend.

Another way to increase parallelism is by increasing the size of the worker pool that queries the cache & backend blocks.

A theoretically ideal value for this config to avoid _any_ queueing would be (Size of blocklist / Max Concurrent Queries).
But also factor in the resources provided to the querier.

## TempoCompactorUnhealthy

Tempo by default uses [Memberlist](https://github.com/hashicorp/memberlist) to persist the ring state between components.
Occasionally this results in old components staying in the ring which particularly impacts compactors because they start
falling behind on the blocklist.  If this occurs port-forward to 3100 on a compactor and bring up `/compactor/ring`.  Use the
"Forget" button to drop any unhealthy compactors.

Note that this more of an art than a science: https://github.com/grafana/tempo/issues/142

## TempoDistributorUnhealthy

Tempo by default uses [Memberlist](https://github.com/hashicorp/memberlist) to persist the ring state between components.
Occasionally this results in old components staying in the ring which does not impact distributors directly, but at some point 
your components will be passing around a lot of unnecessary information. It may also indicate that a component shut down
unexpectedly and may be worth investigating. If this occurs port-forward to 3100 on a distributor and bring up `/distributor/ring`. 
Use the "Forget" button to drop any unhealthy distributors.

Note that this more of an art than a science: https://github.com/grafana/tempo/issues/142

## TempoCompactionsFailing
## TempoFlushesFailing

Check ingester logs for flushes and compactor logs for compations.  Failed flushes or compactions could be caused by any number of
different things.  Permissions issues, rate limiting, failing backend, ...  So check the logs and use your best judgement on how to
resolve.

In the case of failed compactions your blocklist is now growing and you may be creating a bunch of partially written "orphaned"
blocks.  An orphaned block is a block without a `meta.json` that is not currently being created.  These will be invisible to
Tempo and will just hang out forever (or until a bucket lifecycle policy deletes them).  First, resolve the issue so that your 
compactors can get the blocklist under control to prevent high query latencies.  Next try to identify any "orphaned" blocks and
remove them.

In the case of failed flushes your local WAL disk is now filling up.  Tempo will continue to retry sending the blocks
until it succeeds, but at some point your WAL files will start failing to write due to out of disk issues.  If the problem 
persists consider killing the block that's failing to upload in `/var/tempo/wal` and restarting the ingester.

## TempoPollsFailing

If polls are failing check the component that is raising this metric and look for any obvious logs that may indicate a quick fix.
If you see logs about the number of blocks being too long for the job queue then raise the `storage.traces.pool.max_workers` value
to compensate.

Generally, failure to poll just means that the component is not aware of the current state of the backend but will continue working 
otherwise.  Queriers, for instance, will start returning 404s as their internal representation of the backend grows stale.