---
description: "Learn how tracing data can help you understand application insights and performance as well as triaging issues in your services and applications."
keywords:
  - Grafana
  - traces
  - tracing
title: Diagnose errors with traces
menuTitle: Diagnose errors with traces
weight: 400
---

# Diagnose errors with traces

Traces allow you to quickly diagnose errors in your application, ensuring that you can perform Root Cause Analysis (RCA) on request failures.
Trace visualizations help you determine the spans in which errors occur, along with the context behind those errors, leading to a lower mean time to repair (MTTR).

## Meet Handy Site Corp

Handy Site Corp, a fake website company, runs an ecommerce application that includes user authentication, a product catalog, order management, payment processing, and other services.

Handy Site’s operations team receives several alerts for their error SLO for monitored endpoints in their services. Using their Grafana dashboards, they notice that there are several issues. The dashboard provides the percentages of errors:

![Dashboard showing errors in services](/media/docs/tempo/intro/traces-error-SLO.png)

More than 5% of the requests from users are resulting in an error across several endpoints, such as `/beholder`, `/owlbear`, and `/illithid`, which causes a degradation in performance and usability.
It’s imperative for the operations team at Handy Site to quickly troubleshoot the issue.

## Use TraceQL to query data

Tempo has a traces-first query language, [TraceQL](https://grafana.com/docs/tempo/latest/traceql/), that provides a unique toolset for selecting and searching tracing data. TraceQL can match traces based on span and resource attributes, timing, and duration and provides basic aggregate functions.

Handy Site’s services and applications are instrumented for tracing, so they can use TraceQL as a debugging tool. Using three TraceQL queries, the team identifies and validates the root cause of the issue.

### Find HTTP errors

The top-level service, `mythical-requester`, calls other services and deals with requests and responses from their users.
Using Grafana Explore, the operations team starts with a simple TraceQL query to find all traces from this top-level service, where an HTTP response being sent back to a user is 400 or above, such as `Forbidden`, `Not found`, and other responses that don't return data the user expected.

```traceql
{ resource.service.name = "mythical-requester" && span.http.status_code >= 400 } | select(span.http.target)
```

The `select` statement in this query includes an additionally returned field for each matching trace span, namely the value of the `http.target` attribute, the endpoints for the SaaS service.

![Query results showing http.target attribute](/media/docs/tempo/intro/traceql-http-target-handy-site.png)

### Pinpoint the error

The immediate concern is the errors returning an `HTTP 500` status code, which are internal server errors.
Something is occurring in the application, which in turn prevents valid responses to their users’ requests.
This affects the operation team’s SLO error budget, and affects profitability overall for Handy Site.

The team decides to use structural operators to follow an error chain from the `mythical-requester` service to any descendant spans that also have an error status.
Descendant spans can be any span that's descended from the parent span, such as a child or a further child at any depth.
Using this query, the team can pinpoint the downstream service that might be causing the issue.

```traceql
{ resource.service.name = "mythical-requester" && span.http.status_code = 500 } >> { status = error }
```

![TraceQL results showing expanded span](/media/docs/tempo/intro/traceql-error-insert-handy-site.png)

Expanding the erroring span found in the `mythical-server` service shows the team that there is a problem with the data being inserted into the database.
Specifically, that the service is passing a null value for a column in a database table where null values are invalid.

![Error span for INSERT](/media/docs/tempo/intro/traceql-insert-postgres-handy-site.png)

### Verify root cause

After identifying the specific cause of this internal server error,
the team decides to find out if there are any other errors in operations other than `INSERT`s.
Their updated query uses a negated regular expression to find any matches where the database statement either doesn’t exist, or doesn’t start with an `INSERT` clause.
This should expose any other issues causing an internal server error.

```traceql
{ resource.service.name = "mythical-requester" && span.http.status_code = 500 } >> { status = error && span.db.statement !~ "INSERT.*" }
```

This query yields no results, meaning that the root cause of the issues the operations team are seeing is clearly the erroring database insertion call.
At this point, they can swap out the underlying service for a known working version, or deploy a fix to ensure that null data being passed to the service is rejected appropriately.
Requests can be responded to quickly and the issue can be resolved.

![Empty query results](/media/docs/tempo/intro/traceql-no-results-handy-site.png)
