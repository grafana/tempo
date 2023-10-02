---
aliases:
- /docs/tempo/v1.2.1/configuration/multitenancy/
title: Multitenancy
weight: 6
---

Tempo is a multitenant distributed tracing backend. It supports multitenancy through the use
of a header: `X-Scope-OrgID`. This guide details how to setup multitenancy.

## Multitenancy

If you're interested in setting up multitenancy, please consult the [multitenant example](https://github.com/grafana/tempo/tree/main/example/docker-compose/otel-collector-multitenant)
in the repo. This example uses the following settings to achieve multitenancy in Tempo:

- Configure the OTEL Collector to attach the X-Scope-OrgID header on push:
```
exporters:
  otlp:
    headers:
      x-scope-orgid: foo-bar-baz
```

### Grafana 7.5.x and higher
- Configure the Tempo datasource in Grafana to pass the tenant with the same header.
```
- name: Tempo-Multitenant
  jsonData:
    httpHeaderName1: 'X-Scope-OrgID'
  secureJsonData:
    httpHeaderValue1: 'foo-bar-baz'
```

### Grafana 7.4.x
- Configure the Tempo datasource in Grafana to pass the tenant as a bearer token. This is necessary because it is the only header that Jaeger can be configured to pass to its GRPC plugin.
```
- name: Tempo-Multitenant
  jsonData:
    httpHeaderName1: 'Authorization'
  secureJsonData:
    httpHeaderValue1: 'Bearer foo-bar-baz'
```
- Configure Jaeger Query to pass the bearer token to its backend.
```
--query.bearer-token-propagation=true
```

## Important Notes

- Multitenancy on ingestion is currently [only working](https://github.com/grafana/tempo/issues/495) with GPRC and this may never change. It is strongly recommended to use the OpenTelemetry Collector to support multitenancy as described above.
- The way the read path is configured is temporary and should be much more straightforward once the [tempo-query dependency is removed](https://github.com/grafana/tempo/issues/382).

## Enabling Multitenancy
To enable multitenancy on Tempo backend, simply set the following config value on all Tempo components:
```
multitenancy_enabled: true
```

or from the command line:
```
--multitenancy.enabled=true
```

This option will force all Tempo components to require the `X-Scope-OrgID` header.
