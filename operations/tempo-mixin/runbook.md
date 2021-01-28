# Runbook

This document should help with remediating operational issues in Tempo.

## TempoRequestErrors
## TempoRequestLatency

Aside from obvious errors in the logs the only real lever you can pull here is scaling.  Use the Reads or Writes dashboard 
to identify the component that is struggling and scale it up.  It should be noted that right now quickly scaling the 
Ingester component can cause 404s on traces until they are flushed to the backend.  For safety you may only want to 
scale one per hour.  However, if Ingesters are falling over, it's better to scale fast, ingest successfully and throw 404s 
on query than to have an unstable ingest path.  Make the call!

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