---
title: Multitenancy
---

Tempo is a multitenant distributed tracing backend. It supports multitenancy through the use
of a header: `X-Scope-OrgID`. This guide details how to setup or disable multitenancy.

## Multitenancy

If you're interested in setting up multitenancy, please consult the [multitenant example](https://github.com/grafana/tempo/tree/master/example/docker-compose/docker-compose.multitenant.yaml)
in the repo. This example uses the following settings to achieve multitenancy in Tempo:

- Configure the OTEL Collector to attach the X-Scope-OrgID header on push:
```
exporters:
  otlp:
    headers:
      x-scope-orgid: foo-bar-baz
```
- Configure the Tempo datasource in Grafana to pass the tenant as a bearer token. Yes, this is weird. It works b/c it is the only header that Jaeger can be configured to pass to its GRPC plugin.
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

## Disabling Multitenancy
Most Tempo installations will be single tenant. If this is desired simply set the following config
value on all Tempo components:
```
auth_enabled: false
```

or from the command line:
```
--auth.enabled=false
```

This option will force all Tempo components to ignore the `X-Scope-OrgID` header and use the hardcoded
value of `single-tenant`.
