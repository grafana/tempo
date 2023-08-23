---
title: Tag search
description: Troubleshoot No options found in Grafana tag search
weight: 476
aliases:
- ../operations/troubleshooting/search-tag/
---

# Tag search

An issue occurs while searching for traces in Grafana Explore. The **Service Name** and **Span Name** drop down lists are empty, and there is a `No options found` message.

HTTP requests to Tempo query frontend endpoint at `/api/search/tag/service.name/values` would respond with an empty set.


## Root cause

The introduction of a cap on the size of tags causes this issue.

Configuration parameter `max_bytes_per_tag_values_query` causes the return of an empty result
when a query exceeds the configured value.

## Solutions

There are two main solutions to this issue:

* Reduce the cardinality of tags pushed to Tempo. Reducing the number of unique tag values will reduce the size returned by a tag search query.
* Increase the `max_bytes_per_tag_values_query` parameter in the [overrides]({{< relref "../configuration#overrides" >}}) block of your Tempo configuration to a value as high as 50MB.
