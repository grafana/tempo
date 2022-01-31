---
title: Tag search
weight: 476
---

# Tag search

While searching for traces in Grafana Explore UI: `Service Name` and `Span Name` dropdown lists are empty with a *No options found* message.

HTTP calls to Tempo Query Frontend on `/api/search/tag/service.name/values` would answer an empty set.

Note: this happens on Grafana Tempo 1.3 or higher.

## Root cause

Grafana Tempo 1.3 has introduced another way of looking for tags to be used later in Grafana Explore UI, especially `service.name` and `name` tags that are used in the `Service Name` & `Span Name` dropdown lists respectively.

Queries used to fetch those tags are now submitted to a size limit set with the `max_bytes_per_tag_values_query` parameter (documented here: [Tempo configuration overrides](https://grafana.com/docs/tempo/latest/configuration/#overrides)

If by any chance this query goes over the size configured there: it will return an empty result.

## Resolution

There are two main course of actions to solve this issue:

* either reduce the cardinality of tags pushed to Tempo: reducing the number of unique tag values will reduce the size returned by the tag search query later on.
* increase the `max_bytes_per_tag_values_query` in the override section of your Tempo configuration: there is no rule of thumb though 10Mb or even 50Mb is not unheard of.
