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
Occasionally this results in old components staying in the ring which impacts distributors only if you are using the 
global rate limiting strategy.  If this occurs port-forward to 3100 on a distributor and bring up `/distributor/ring`.  Use the
"Forget" button to drop any unhealthy distributors.

Note that this more of an art than a science: https://github.com/grafana/tempo/issues/142