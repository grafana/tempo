## OpenTelemetry Collector
This example is based on example from [docker-compose.yaml](example/docker-compose/otel-collector/docker-compose.yaml).
Setup shows how to connect application with opentelemetry tracing system with usage of custom docker-compose
network.

1. First start up the stack.

```console
docker-compose up -d
```

2. In your application `compose.yaml` file define:

```yaml

networks:
  tempo-network:
    external:
      name: tempo-network

```
Then you can use this network in services:

```yaml

services:
  backend:
    ...
    environment:
      OTEL_EXPORTER_OTLP_ENDPOINT: http://otel-collector:4317
    networks:
      - tempo-network

```

`backend` service have now access to `otel-collector`. Instructions how to send data to OTLP you can find there:
[link](https://opentelemetry.io/docs/instrumentation/python/exporters/#trace-1).