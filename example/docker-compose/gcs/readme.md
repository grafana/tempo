## GCS

In this example Tempo is configured to write data to GCS via [fake-gcs-server](https://github.com/fsouza/fake-gcs-server) which presents a GCS compatible API.

1. First start up the azure stack.

```console
docker-compose up -d
```

At this point, the following containers should be spun up -

```console
docker-compose ps
```

```
                  Name                                Command               State                        Ports
------------------------------------------------------------------------------------------------------------------------------------
gcs_gcs_1                        /bin/fake-gcs-server -data ...   Up      0.0.0.0:4443->4443/tcp
gcs_grafana_1                    /run.sh                          Up      0.0.0.0:3000->3000/tcp
gcs_prometheus_1                 /bin/prometheus --config.f ...   Up      0.0.0.0:9090->9090/tcp
gcs_synthetic-load-generator_1   ./start.sh                       Up
gcs_tempo_1                      /tempo -config.file=/etc/t ...   Up      0.0.0.0:59543->14268/tcp, 0.0.0.0:3200->3200/tcp
```

2. If you're interested you can kind of see the wal/blocks as they are being created. Navigate to https://localhost:4443/storage/v1/b/tempo/o
   to get a dump of all objects in the bucket.

3. The synthetic-load-generator is now printing out trace ids it's flushing into Tempo. To view its logs use -

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
   Also notice that you can query Tempo metrics from the Prometheus data source setup in Grafana.

5. To stop the setup use -

```console
docker-compose down -v
```
