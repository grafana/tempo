---
Authors: Yuri Shkuro (@yurishkuro), Tom Wilkie (@tomwilkie), Cyril Tovena (@cyriltovena), Martin Disibio (@mdisibio), Joe Elliott (@joe-elliott)
Created: 2021 October - 2022 March
Last updated: 2022-04-12
---

# TraceQL Concepts

## Summary

This design document describes a language for selecting traces that Tempo will implement. This language is currently only focused on trace selection. It will be extended to derive metrics from traces in the future.

This document is **not** meant to be a complete language specification. Rather it is meant to invite the community to comment on the basic structure and concepts of the language.

### Table of Contents

- [Capabilities](#capabilities)
- [Structure](#structure)
- [Selecting spans](#selecting-spans)
  - [Intrinsic fields](#intrinsic-fields)
  - [Attribute fields](#attribute-fields)
  - [Field expressions](#field-expressions)
- [Combining Spansets](#combining-spansets)
  - [Structural Operators](#structural-operators)
- [Aggregators](#aggregators)
- [Pipelining](#pipelining)
- [Grouping](#grouping)
- [Examples](#examples)
  - [Shorthand](#shorthand)

## Capabilities

A TraceQL query can select traces based on 

- span attributes, timing and duration
- structural relationships between spans
- aggregated data from the spans in a trace

The TraceQL language is designed to use the same syntax and semantics as PromQL and LogQL where possible. The syntax and semantics differ due to the specialized needs of querying traces.

## Structure

A query is an expression that is evaluated on one trace at a time. The query is structured as a set of chained
expressions (a pipeline). Each expression in the pipeline selects or discards spansets from being included in the results set. E.g.

`{ .http.status = 200 } | by(.namespace) | count() > 3`

Queries select sets of spans and filter them through a pipeline of aggregators and conditions. If a spanset is produced after evaluation
on a trace then this spanset (and by extension the trace) is included in the resultset of the query.

## Selecting spans

In TraceQL the curly brackets `{}` always select a set of spans from the current trace. They will commonly be paired with a condition to reduce the spans being passed in.

`{ .http.status = 200 }`

This simple query will be evaluated on every span of every trace, one at a time. 

If the trace being evaluated contains no spans with an attribute `http.status` with the value `200` then no spans will be selected and this trace will not appear in our resultset.

If the trace does contain spans with an attribute `http.status` with the value `200` then only those spans will be returned. The trace is reduced to only the set of spans that match the condition inside the `{}`. The resultset will contain only this subset of spans matching the condition.

### Intrinsic fields

Every span has certain intrinsic fields that can be referenced when selecting spans

| field name | description                                 |
| ---------- | ------------------------------------------- |
| `duration` | end - start time of the span                |
| `name`     | operation or span name                      |
| `status`   | status values are error, ok, or unset       |
| `parent`   | the parent of this span                     |

**Examples**
Find traces that contain spans whose duration is greater than 2 seconds:  
`{ duration > 2s }`

Find traces that contain a span named "HTTP POST":  
`{ name = "HTTP POST" }`

### Attribute fields

We can refer to dynamic attributes (also known as tags) on the span or the span's resource.

**Examples**

Find traces with the GET http method:  
`{ .http.method = "GET" }`

Find traces that passed through the prod namespace:  
`{ .namespace = "prod" }`

Find traces that cross service boundaries:  
`{ parent.service.name != .service.name }`

### Scoped attribute fields

Attributes can be specifically scoped to either "span" or "resource". Specifying "span" or "resource"
can result in significant performance benefits.

`{ span.http.status = 200 }`
`{ resource.namespace = "prod" }`

### Field expressions

Fields can also be combined in various expected ways.

**Examples**

Find traces where the difference of the duration of a set of parent/child spans exceeds 500ms:  
`{ parent.duration - duration > 500ms }`

Find traces with "success" http status codes:  
`{ .http.status >= 200 && .http.status < 300 }`

Note that the second expression requires both conditions to be true on the same span. The entire expression inside of `{}` must be evaluated as true on a single span for it to be included in the resultset.

## Combining Spansets

Logical operators combine sets of spans. For example, if we wanted to find a trace that went through two specific regions:  
`{ .region = "eu-west-0" } && { .region = "eu-west-1" } `

Note the difference between the above and the following:  
`{ .region = "eu-west-0"  && .region = "eu-west-1" } `

The second expression will return no traces because it's impossible for both conditions to be simultaneously true on the same span.

### Structural operators

Structural operators evaluate traces based on the span relationships within the span tree. 

`{ } >> { }`  
  This is the descendant operator. The spans returned from this operator will match the right hand side conditions while also being descendants of spans that match the left hand side conditions.

`{ } > { }`  
  This is the child operator. The spans returned from this operator will match the right hand side conditions while also being children of spans that match the left hand side conditions.

`{ } ~ { }`  
  This is the sibling operator. The spans returned from this operator will match the right hand side conditions while also being siblings of spans that match the left hand side conditions.

## Aggregators

All of the above expressions involve asking questions about individual spans. However, sometimes we want to ask questions about a set of spans. For that we have aggregate functions.

**Examples**

Find traces where the total number of spans is greater than 10:  
`count() > 10`

Find traces where the average duration is greater than 1s:  
`avg(duration) > 1s`

## Expression Pipelining

Pipelining allows us to "pipe" a set of spans from one expression to the next. This is particularly useful if you want to perform an aggregate over a subset of a trace.

**Examples**

Find traces that have more than 3 spans with an attribute `http.status` with a value of `200`:  
`{ .http.status = 200 } | count() > 3`

## Grouping

Grouping allows us to take a trace and break it down into sets of spans that are evaluated by future pipeline entries. Each set of spans created by the group is _individually_ evaluated by downstream expressions.

**Examples**

Find traces that have more than 5 spans in any region:  
`by(.region) | count() > 5 `

## Examples

Any span matches an attribute:  
`{ .namespace = "prod" }`

Two attributes appear on the same span:  
`{ .namespace = "prod" && .http.status = 200 }`

Two attributes appear anywhere within a trace:  
`{ .namespace = "prod" } && { .http.status = 200 }`

Any span has a duration over one second:  
`{ duration > 1s }`

The trace as a whole has a duration of over one second:  
`max(end) - min(start) > 1s`

The average duration of spans exceeds 1s and any span has a specific attribute:
`avg(duration) > 1s && { .namespace = "prod" }`

A trace has over 5 spans with http.status = 200 in any given namespace:  
`{ .http.status = 200 } | by(.namespace) | count() > 5`

A trace passed through two regions in a specific order:  
`{ .region = "eu-west-0" } >> { .region = "eu-west-1" }`

A trace sees network latency > 1s when passing between any two services:  
`{ parent.service.name != .service.name } | max(parent.duration - duration) > 1s`
