---
title: Too many requests error
description: Troubleshoot Too many requests error for a Tempo query
weight: 480
aliases:
---

# Too many requests (429 error code)

if an issue occurs during a Tempo query, the error response may look like:

```
429 failed to execute TraceQL query: {resource.service.name != nil} | rate() by(resource.service.name) Status: 429 Too Many Requests Body: job queue full
```


## Root cause

Tempo parallelizes work by breaking a single query into multiple requests (jobs) that are distributed to the queriers. 
Increasing the time range results in more jobs being created.
To ensure fair resource usage and to prevent the "noisy neighbor" problem in multi-tenant environments, Tempo limits the number of jobs a tenant can run concurrently. The limit of maximun number of jobs per tenant is controlled by the query-frontend value  `max_outstanding_per_tenant`.

## Solutions

There are two main solutions to this issue:

* Reduce the time range of the query.
* Increase the `max_outstanding_per_tenant` parameter in the query-frontend configuration from the default of 2000 jobs.

```yaml
query-frontend:
  max_outstanding_per_tenant:: <max number of jobs>
```
