---
title: Overview
draft: true
---

Grafana Tempo is a highly robust, minimal dependency distributed tracing backend that stores and retrieves traces by id only.

## Multi Tenancy

Tempo is natively multi-tenant. Multi tenant behavior is implemented through the use of
a HTTP header `X-Scope-OrgID`.

## Modes of Operation

Tempo can be operated in Single binary as well as Microservices mode.
