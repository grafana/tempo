---
description: "Learn how tracing data can help you understand application insights and performance as well as triaging issues in your services and applications."
keywords:
  - Grafana
  - traces
  - tracing
title: Diagnose errors with traces
menuTitle: Diagnose errors with traces
weight: 120
---

# Diagnose errors with traces

Traces are especially powerful when:

* Identify cause of bottlenecks using application insights and performance
* Diagnose 500 errors and reduce MTTR

Each use case provides real-world examples, including the background of the use case and how tracing highlighted and helped resolve any issues.

Handy Site’s operations team receives several alerts on errors for monitored endpoints in their services. Using their Grafana dashboards, they notice that there are several issues. The dashboard provides the percentages of errors:

SCREENSHOT

More than 5% of the requests from users are resulting in an error across several endpoints, which causes a degradation in performance and usability. It’s imperative for the operations team at Handy Site to quickly troubleshoot the issue.

## Use TraceQL to query data

Tempo has a traces-first query language, [TraceQL](https://grafana.com/docs/tempo/latest/traceql/), that provides a unique toolset for selecting and searching tracing data. TraceQL can match traces based on span and resource attributes, timing, and duration and provides basic aggregate functions.

Handy Site’s services and applications are instrumented for tracing, so they can use TraceQL as a debugging tool. Using three TraceQL queries, the team identifies and validates the root cause of the issue.

The top-level service, `mythical-requester`, calls other services and deals with requests and responses from their users.

Using Grafana Explore, the operations team starts with a simple TraceQL query to find all traces from this top-level service, where an HTTP response being sent back to a user is 400 or above, such as ‘Forbidden’, ‘Not found’, and other responses that do not return data the user expected.

```traceql
{ resource.service.name = "mythical-requester" && span.http.status_code >= 400 } | select(span.http.target)
```

The `select` statement in this query includes an additionally returned field for each matching trace span, namely the value of the `http.target` attribute, the endpoints for the SaaS service.

SCREENSHOT

By expanding out the returned services, the operations team examines the HTTP status codes associated with the matching spans.
They see that there is a 404 error, a `Not Found` on the `/debug/pprof/block` endpoint, which is interesting because that’s the endpoint for a profiler to scrape data from.
This won’t be an issue with application endpoints serving users, but they note that the profiler won’t be receiving data from the service and should be looked at later.

The immediate concern is the errors returning an HTTP 500 status code, which are internal server errors.
Something is occurring in the application, which in turn prevents valid responses to their users’ requests.
This affects the operation team’s SLO error budget, and affects profitability overall for Handy Site.

The team decides to use structural operators to follow an error chain from the mythical-requester service to any descendant spans that also have an error status.
Descendant span can be any span that is descended from the parent span, such as a child or a further child at any depth.
Using this query, the team can pinpoint the downstream service that might be causing the issue.

```traceql
{ resource.service.name = "mythical-requester" && span.http.status_code = 500 } >> { status = error }
```

SCREENSHOT


Expanding the erroring span found in the `mythical-server` service shows the team that there is a problem with the data being inserted into the database.
Specifically, that the service is passing a null value for a column in a database table where null values are invalid.

SCREENSHOT

After identifying the specific cause of the internal server error, the team rewrites the TraceQL query to focus on the database statement `INSERT`.
Their updated query uses a negated regular expression to find any matches where the database statement either doesn’t exist, or doesn’t start with an `INSERT` clause.
This should expose any other issues causing an internal server error.

```traceql
{ resource.service.name = "mythical-requester" && span.http.status_code = 500 } >> { status = error && span.db.statement !~ "INSERT.*"}
```

SCREENSHOT

This query yields no results, meaning that the root cause of the issues the operations team are seeing is clearly the erroring database insertion call.
At this point, they can swap out the underlying service for a known working version, or deploy a fix to ensure that null data being passed to the service is rejected appropriately.
Requests will now be responded to quickly and the issue will be resolved.
