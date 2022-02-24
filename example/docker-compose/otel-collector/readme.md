## OpenTelemetry Collector
This example highlights setting up the OpenTelemetry Collector in a simple tracing pipeline.

1. First start up the stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```
```
                  Name                                Command               State           Ports
----------------------------------------------------------------------------------------------------------
otel-collector_grafana_1                    /run.sh                          Up      0.0.0.0:3000->3000/tcp
otel-collector_otel-collector_1             /otelcol --config=/etc/ote ...   Up      55678/tcp, 55679/tcp
otel-collector_synthetic-load-generator_1   ./start.sh                       Up
otel-collector_tempo_1                      /tempo -config.file=/etc/t ...   Up      0.0.0.0:59538->14268/tcp, 0.0.0.0:59540->3200/tcp,
                                                                                    0.0.0.0:59537->4317/tcp, 0.0.0.0:59536->4318/tcp,
                                                                                    0.0.0.0:59539->9411/tcp
```

2. If you're interested you can see the wal/blocks as they are being created.

```console
ls tempo-data/
```

3. The synthetic-load-generator is now printing out trace ids it's flushing into Tempo.  To view its logs use -

```console
docker-compose logs -f synthetic-load-generator
```
```
synthetic-load-generator_1  | 20/10/24 08:26:55 INFO ScheduledTraceGenerator: Emitted traceId 48367daf25266daa for service frontend route /currency
synthetic-load-generator_1  | 20/10/24 08:26:55 INFO ScheduledTraceGenerator: Emitted traceId 10e50d2aca58d5e7 for service frontend route /cart
synthetic-load-generator_1  | 20/10/24 08:26:55 INFO ScheduledTraceGenerator: Emitted traceId 51a4ac1638ee4c63 for service frontend route /shipping
synthetic-load-generator_1  | 20/10/24 08:26:55 INFO ScheduledTraceGenerator: Emitted traceId 1219370c6a796a6d for service frontend route /product
```

Logs are in the form

```
Emitted traceId <traceid> for service frontend route /cart
```

Copy one of these trace ids.

4. Navigate to [Grafana](http://localhost:3000/explore) and paste the trace id to request it from Tempo.

5. To stop the setup use -

```console
docker-compose down -v
```
